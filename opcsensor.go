// Package customsensor implements a sensor where all methods are unimplemented.
// It extends the built-in resource subtype sensor and implements methods to handle resource construction and attribute configuration.

package opcsensor

import (
	"context"
	"errors"
	"io"
	"time"

	"go.viam.com/rdk/components/sensor"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"

	"go.viam.com/utils"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

var (
	Model            = resource.NewModel("viam-soleng", "opc-ua", "opcsensor")
	errUnimplemented = errors.New("unimplemented")
)

func init() {
	resource.RegisterComponent(sensor.API, Model,
		resource.Registration[sensor.Sensor, *Config]{
			Constructor: newOPCSensor,
		},
	)
}

// TODO: Change the Config struct to contain any values that you would like to be able to configure from the attributes field in the component
// configuration. For more information see https://docs.viam.com/build/configure/#components
type Config struct {
	Endpoint string   `json:"endpoint"`
	NodeIDs  []string `json:"nodeids"`
}

// Validate validates the config and returns implicit dependencies.
// TODO: Change the Validate function to validate any config variables.
func (cfg *Config) Validate(path string) ([]string, error) {
	// OPC config validation
	if cfg.Endpoint == "" {
		return nil, utils.NewConfigValidationFieldRequiredError(path, "endpoint")
	}

	// TODO: return implicit dependencies if needed as the first value
	return []string{}, nil
}

// Constructor for a custom sensor that creates and returns a customSensor.
// TODO: update the customSensor struct and the initialization.
func newOPCSensor(ctx context.Context, deps resource.Dependencies, rawConf resource.Config, logger logging.Logger) (sensor.Sensor, error) {
	// This takes the generic resource.Config passed down from the parent and converts it to the
	// model-specific (aka "native") Config structure defined above, making it easier to directly access attributes.
	conf, err := resource.NativeConfig[*Config](rawConf)
	if err != nil {
		return nil, err
	}

	// Create a cancelable context for custom sensor
	cancelCtx, cancelFunc := context.WithCancel(context.Background())

	s := &opcSensor{
		name:       rawConf.ResourceName(),
		logger:     logger,
		cfg:        conf,
		cancelCtx:  cancelCtx,
		cancelFunc: cancelFunc,
	}

	// TODO: If your custom component has dependencies, perform any checks you need to on them.

	// The Reconfigure() method changes the values on the customSensor based on the attributes in the component config
	if err := s.Reconfigure(ctx, deps, rawConf); err != nil {
		logger.Error("Error configuring module with ", rawConf)
		return nil, err
	}

	return s, nil
}

// TODO: update the opcSensor struct with any fields you require.
type opcSensor struct {
	name   resource.Name
	logger logging.Logger
	cfg    *Config

	cancelCtx  context.Context
	cancelFunc func()

	// OPC client
	opcclient *opcua.Client
}

func (s *opcSensor) Name() resource.Name {
	return s.name
}

// Reconfigures the model. Most models can be reconfigured in place without needing to rebuild. If you need to instead create a new instance of the sensor, throw a NewMustBuildError.
func (s *opcSensor) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	cfg, err := resource.NativeConfig[*Config](conf)
	if err != nil {
		s.logger.Warn("Error reconfiguring module with ", err)
		return err
	}

	s.name = conf.ResourceName()

	// Update nodeIDs
	s.cfg.NodeIDs = cfg.NodeIDs

	// OPC client init
	// examples: https://github.com/gopcua/opcua/blob/main/examples/read/read.go
	opcclient, err := opcua.NewClient(s.cfg.Endpoint, opcua.SecurityMode(ua.MessageSecurityModeNone))
	if err != nil {
		s.logger.Error("OPC client not initiated: ", err)
		return err
	}
	s.opcclient = opcclient
	return nil
}

// Read sensor values
func (s *opcSensor) Readings(ctx context.Context, extra map[string]interface{}) (map[string]interface{}, error) {
	// TODO: Obtain and return readings.
	readResponse, err := s.readOPC(ctx)
	if err != nil {
		return nil, err
	}

	result := map[string]interface{}{}
	for idx, val := range readResponse.Results {
		result[s.cfg.NodeIDs[idx]] = val.Value.Value()
	}

	return result, nil
}

// DoCommand is a place to add additional commands to extend the sensor API. This is optional.
func (s *opcSensor) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	s.logger.Error("DoCommand method unimplemented")
	return nil, errUnimplemented
}

// Close closes the underlying generic.
func (s *opcSensor) Close(ctx context.Context) error {
	s.cancelFunc()
	s.opcclient.Close(ctx)
	return nil
}

func (s *opcSensor) readOPC(ctx context.Context) (*ua.ReadResponse, error) {

	if err := s.opcclient.Connect(ctx); err != nil {
		s.logger.Fatal(err)
	}
	defer s.opcclient.Close(ctx)
	s.logger.Info("OPC client successfully connected to: ", s.cfg.Endpoint)

	var readIDs []*ua.ReadValueID

	for _, nodeID := range s.cfg.NodeIDs {
		id, err := ua.ParseNodeID(nodeID)
		if err != nil {
			s.logger.Fatalf("invalid node id: %v", err)
			return nil, err
		}
		readIDs = append(readIDs, &ua.ReadValueID{NodeID: id})
	}

	req := &ua.ReadRequest{
		MaxAge:             2000,
		NodesToRead:        readIDs,
		TimestampsToReturn: ua.TimestampsToReturnBoth,
	}

	var resp *ua.ReadResponse
	for {
		var err error
		resp, err = s.opcclient.Read(ctx, req)
		if err == nil {
			break
		}

		// Following switch contains known errors that can be retried by the user.
		// Best practice is to do it on read operations.
		switch {
		case err == io.EOF && s.opcclient.State() != opcua.Closed:
			// has to be retried unless user closed the connection
			time.After(1 * time.Second)
			continue

		case errors.Is(err, ua.StatusBadSessionIDInvalid):
			// Session is not activated has to be retried. Session will be recreated internally.
			time.After(1 * time.Second)
			continue

		case errors.Is(err, ua.StatusBadSessionNotActivated):
			// Session is invalid has to be retried. Session will be recreated internally.
			time.After(1 * time.Second)
			continue

		case errors.Is(err, ua.StatusBadSecureChannelIDInvalid):
			// secure channel will be recreated internally.
			time.After(1 * time.Second)
			continue

		default:
			s.logger.Fatalf("Read failed: %s", err)
		}
	}

	if resp != nil && resp.Results[0].Status != ua.StatusOK {
		s.logger.Fatalf("Status not OK: %v", resp.Results[0].Status)
	}

	// DEBUGGING
	/*
		out, err := json.Marshal(resp)
		if err != nil {
			panic(err)
		}

		//resultString := fmt.Sprintf("%#v", resp)
		s.logger.Infof("Results: %v", string(out))
	*/
	return resp, nil

}
