package cubefs

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/majlu/my-cubefs-csi/pkg/util"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

const (
	KVolumeName = "volName"
	KMasterAddr = "masterAddr"
	KLogLevel   = "logLevel"
	KLogDir     = "logDir"
	KOwner      = "owner"
	KConsulAddr = "consulAddr"
	KVolType    = "volType"
	KMountPoint = "mountPoint"
)

const (
	defaultClientConfPath = "/cfs/conf/"
	defaultLogDir         = "/cfs/logs/"
	defaultLogLevel       = "info"
	jsonFileSuffix        = ".json"
	defaultConsulAddr     = "http://consul-service.cubefs.svc.cluster.local:8500"
	defaultVolType        = "0"
)
const (
	ErrCodeVolNotExists = 7

	ErrDuplicateVolMsg = "duplicate vol"
)

type CfsServer struct {
	clientConfFile string
	masterAddrs    []string
	clientConf     map[string]string
}

// Create and Delete Volume Response
type cfsServerResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data string `json:"data,omitempty"`
}

func NewCfsServer(volName string, param map[string]string) (cs *CfsServer, err error) {
	masterAddr := param[KMasterAddr]
	if len(volName) == 0 || len(masterAddr) == 0 {
		return nil, fmt.Errorf("invalid argument for initializing cfsServer")
	}

	newVolName := getValueWithDefault(param, KVolumeName, volName)
	clientConfFile := defaultClientConfPath + newVolName + jsonFileSuffix
	// Owner ID can be a random string
	newOwner := util.ShortenString(fmt.Sprintf("csi_%d", time.Now().UnixNano()), 20)
	param[KMasterAddr] = masterAddr
	param[KVolumeName] = newVolName
	param[KOwner] = getValueWithDefault(param, KOwner, newOwner)
	param[KLogLevel] = getValueWithDefault(param, KLogLevel, defaultLogLevel)
	param[KLogDir] = defaultLogDir + newVolName
	// Consul address may be no effect if storage class is not set the param
	param[KConsulAddr] = getValueWithDefault(param, KConsulAddr, defaultConsulAddr)
	param[KVolType] = getValueWithDefault(param, KVolType, defaultVolType)
	return &CfsServer{
		clientConfFile: clientConfFile,
		masterAddrs:    strings.Split(masterAddr, ","),
		clientConf:     param,
	}, err
}

func (cs *CfsServer) createVolume(capacityGB int64) (err error) {
	valName := cs.clientConf[KVolumeName]
	owner := cs.clientConf[KOwner]
	volType := cs.clientConf[KVolType]

	return cs.forEachMasterAddr("CreateVolume", func(addr string) error {
		url := fmt.Sprintf("http://%s/admin/createVol?name=%s&capacity=%v&owner=%v&volType=%v",
			addr, valName, capacityGB, owner, volType)
		klog.InfoS("createVol url", "url", url)
		resp, err := cs.executeRequest(url)
		if err != nil {
			return err
		}

		if resp.Code != 0 {
			if strings.Contains(resp.Msg, ErrDuplicateVolMsg) {
				klog.InfoS("duplicate to create volume. ", "url", url, "msg", resp.Msg)
				return nil
			}

			return fmt.Errorf("create volume failed: url(%v) code=(%v), msg: %v", url, resp.Code, resp.Msg)
		}

		return nil
	})
}

func (cs *CfsServer) executeRequest(url string) (*cfsServerResponse, error) {
	httpResp, err := http.Get(url)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "request url failed, url(%v) err(%v)", url, err)
	}

	defer httpResp.Body.Close()
	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "read http response body, url(%v) bodyLen(%v) err(%v)", url, len(body), err)
	}

	resp := &cfsServerResponse{}
	if err = json.Unmarshal(body, resp); err != nil {
		return nil, status.Errorf(codes.Unavailable, "unmarshal http response body, url(%v) msg(%v) err(%v)", url, resp.Msg, err)
	}
	return resp, nil
}

func (cs *CfsServer) forEachMasterAddr(stage string, f func(addr string) error) (err error) {
	for _, addr := range cs.masterAddrs {
		if err = f(addr); err == nil {
			break
		}
		klog.ErrorS(err, "try master addr failed", "stage", stage, "addr", addr)
	}

	if err != nil {
		klog.ErrorS(err, "Failed with all masters", "stage", stage)
		return err
	}

	return nil
}

func getValueWithDefault(param map[string]string, key string, defaultValue string) string {
	value := param[key]
	if len(value) == 0 {
		value = defaultValue
	}

	return value
}

func (cs *CfsServer) persistClientConf(mountPoint string) error {
	cs.clientConf[KMountPoint] = mountPoint
	_ = os.Mkdir(cs.clientConf[KLogDir], 0777)
	clientConfBytes, _ := json.Marshal(cs.clientConf)
	err := os.WriteFile(cs.clientConfFile, clientConfBytes, 0444)
	if err != nil {
		return status.Errorf(codes.Internal, "create client config file fail. err: %v", err.Error())
	}

	klog.InfoS("Create client config file success", "volumeId", cs.clientConf[KVolumeName])
	return nil
}

func (cs *CfsServer) runClient() error {
	return MountVolume(cs.clientConfFile)
}

func (cs *CfsServer) deleteVolume() (err error) {
	ownerMd5, err := cs.getOwnerMd5()
	if err != nil {
		return err
	}

	valName := cs.clientConf[KVolumeName]
	return cs.forEachMasterAddr("DeleteVolume", func(addr string) error {
		url := fmt.Sprintf("http://%s/vol/delete?name=%s&authKey=%v", addr, valName, ownerMd5)
		klog.InfoS("deleteVol url", "url", url)
		resp, err := cs.executeRequest(url)
		if err != nil {
			return err
		}

		if resp.Code != 0 {
			if resp.Code == ErrCodeVolNotExists {
				klog.InfoS("volume not exists, assuming the volume has already been deleted.",
					"volName", valName, "respCode", resp.Code, "msg", resp.Msg)
				return nil
			}
			return fmt.Errorf("delete volume[%s] is failed. code:%v, msg:%v", valName, resp.Code, resp.Msg)
		}

		return nil
	})
}

func (cs *CfsServer) getOwnerMd5() (string, error) {
	owner := cs.clientConf[KOwner]
	key := md5.New()
	if _, err := key.Write([]byte(owner)); err != nil {
		return "", status.Errorf(codes.Internal, "calc owner[%v] md5 fail. err(%v)", owner, err)
	}

	return hex.EncodeToString(key.Sum(nil)), nil
}
