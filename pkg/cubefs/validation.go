package cubefs

import "fmt"

func ValidateDriverOptions(options *Options) error {
	if err := validateMode(options.Mode); err != nil {
		return fmt.Errorf("Invalid mode: %w", err)
	}

	return nil
}

func validateMode(mode Mode) error {
	if mode != AllMode && mode != ControllerMode && mode != NodeMode {
		return fmt.Errorf("Mode is not supported (actual: %s, supported: %v)", mode, []Mode{AllMode, ControllerMode, NodeMode})
	}

	return nil
}
