// Package opcsensor implements a sensor which allows reading opc ua nodes through the readings api and write attributes through do_command

package opcsensor

import (
	"container/list"
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"go.viam.com/rdk/components/sensor"
	"go.viam.com/rdk/data"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"

	"go.viam.com/utils"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

var (
	Model = resource.NewModel("viam-soleng", "opc-ua", "opcsensor")
)

func init() {
	resource.RegisterComponent(sensor.API, Model,
		resource.Registration[sensor.Sensor, *Config]{
			Constructor: newOPCSensor,
		},
	)
}

// OPC UA client configuration
type Config struct {
	Endpoint  string   `json:"endpoint"`
	Subscribe string   `json:"subscribe"`
	NodeIDs   []string `json:"nodeids"`
}

// Validate validates the config and returns implicit dependencies.
func (cfg *Config) Validate(path string) ([]string, error) {
	// OPC config validation
	if cfg.Endpoint == "" {
		return nil, utils.NewConfigValidationFieldRequiredError(path, "endpoint")
	}

	// TODO: return implicit dependencies if needed as the first value
	return []string{}, nil
}

// Constructor for a custom sensor that creates and returns an opcsensor.
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
	mu     sync.Mutex

	cancelCtx  context.Context
	cancelFunc func()

	// OPC client
	opcclient       *opcua.Client
	opcSubscription *opcua.Subscription
	notifyChannel   chan *opcua.PublishNotificationData
	queue           list.List
	queueEmpty      bool
}

func (s *opcSensor) Name() resource.Name {
	return s.name
}

// Reconfigures the model. Most models can be reconfigured in place without needing to rebuild. If you need to instead create a new instance of the sensor, throw a NewMustBuildError.
func (s *opcSensor) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {

	s.mu.Lock()
	defer s.mu.Unlock()

	cfg, err := resource.NativeConfig[*Config](conf)
	if err != nil {
		s.logger.Warn("Error reconfiguring module with ", err)
		return err
	}

	s.name = conf.ResourceName()
	s.logger.Infof("name: %s", s.name)
	// Update nodeIDs
	s.cfg.NodeIDs = cfg.NodeIDs
	s.logger.Infof("nodeids: %v", s.cfg.NodeIDs)
	// Update subscription
	s.cfg.Subscribe = cfg.Subscribe
	s.logger.Infof("subscribe: %s", s.cfg.Subscribe)

	// OPC client init
	// examples: https://github.com/gopcua/opcua/blob/main/examples/read/read.go
	opcclient, err := opcua.NewClient(s.cfg.Endpoint, opcua.SecurityMode(ua.MessageSecurityModeNone))
	if err != nil {
		s.logger.Error("OPC client not initiated: ", err)
		return err
	}
	s.opcclient = opcclient
	if err := s.opcclient.Connect(ctx); err != nil {
		s.logger.Error(err)
		return err
	}
	s.logger.Infof("OPC client successfully connected to: %s", s.cfg.Endpoint)

	// Create OPC UA subscription if set in config
	if s.cfg.Subscribe == "data" {
		//interval := 100 * time.Millisecond // event publishing frequency
		params := &opcua.SubscriptionParameters{}
		s.notifyChannel = make(chan *opcua.PublishNotificationData)
		s.opcSubscription, err = s.opcclient.Subscribe(ctx, params, s.notifyChannel)
		if err != nil {
			return err
		}
		s.logger.Infof("OPC subscription created: %v", s.opcSubscription.SubscriptionID)

		// Parse configured node ids and create a list of items to be monitored
		var miCreateRequests []*ua.MonitoredItemCreateRequest
		for _, nodeID := range s.cfg.NodeIDs {
			id, err := ua.ParseNodeID(nodeID)
			if err != nil {
				s.logger.Errorf("invalid node id: %v", err)
				return err
			}
			miCreateRequests = append(miCreateRequests, valueRequest(id))
		}
		// Add subscriptions
		res, err := s.opcSubscription.Monitor(ctx, ua.TimestampsToReturnBoth, miCreateRequests...)
		if err != nil || res.Results[0].StatusCode != ua.StatusOK {
			s.logger.Error(err)
		}

		// Start monitoring nodeids for data changes
		go s.monitorData()
	}

	return nil
}

// Read and return sensor values
func (s *opcSensor) Readings(ctx context.Context, extra map[string]interface{}) (map[string]interface{}, error) {

	var readResponse *ua.ReadResponse
	var err error

	if s.cfg.Subscribe == "data" {
		if extra[data.FromDMString] != true {
			// Read most recent values from queue
			return map[string]interface{}{"value": s.queue.Back()}, nil
		}
		if s.queueEmpty {
			return nil, data.ErrNoCaptureToStore
		} else {
			value := s.queue.Front()
			if s.queue.Len() == 1 {
				s.queueEmpty = true
			} else {
				s.queue.Remove(value)
			}
			return map[string]interface{}{"value": value}, nil
		}

	} else {
		// Request values directly from OPC server
		readResponse, err = s.readData(ctx)
		if err != nil {
			return nil, err
		}
	}

	result := map[string]interface{}{}
	for idx, val := range readResponse.Results {
		result[s.cfg.NodeIDs[idx]] = val.Value.Value()
	}
	return result, nil
}

// DoCommand is used to set opc ua attributes
func (s *opcSensor) DoCommand(ctx context.Context, cmd map[string]interface{}) (map[string]interface{}, error) {
	if wvs, ok := cmd["write"]; ok {
		if nodes, ok := wvs.(map[string]any); ok {
			nodesToWrite := []*ua.WriteValue{}
			for k, v := range nodes {
				// Check NodeID
				nodeid, err := ua.ParseNodeID(k)
				if err != nil {
					s.logger.Errorf("invalid node id: %v", err)
					return nil, err
				}
				// Convert value
				v, err := ua.NewVariant(v)
				if err != nil {
					s.logger.Errorf("invalid value: %v", err)
					return nil, err
				}

				nwv := ua.WriteValue{
					NodeID:      nodeid,
					AttributeID: ua.AttributeIDValue,
					Value: &ua.DataValue{
						EncodingMask: ua.DataValueValue,
						Value:        v,
					},
				}
				nodesToWrite = append(nodesToWrite, &nwv)
			}
			writeRequest := &ua.WriteRequest{
				NodesToWrite: nodesToWrite,
			}
			resp, err := s.opcclient.Write(ctx, writeRequest)
			if err != nil {
				return nil, err
			} else {
				result := []any{}
				for _, code := range resp.Results {
					result = append(result, ua.StatusCodes[code])
				}
				return map[string]interface{}{"results": result}, nil
			}
		}
	}
	return nil, nil
}

// Close closes the underlying generic.
func (s *opcSensor) Close(ctx context.Context) error {
	s.cancelFunc()
	s.opcSubscription.Cancel(ctx)
	s.opcclient.Close(ctx)
	return nil
}

// TODO: Rename to readData
func (s *opcSensor) readData(ctx context.Context) (*ua.ReadResponse, error) {
	var readIDs []*ua.ReadValueID

	for _, nodeID := range s.cfg.NodeIDs {
		id, err := ua.ParseNodeID(nodeID)
		if err != nil {
			s.logger.Errorf("invalid node id: %v", err)
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
			s.logger.Errorf("Read failed: %s", err)
		}
	}

	if resp != nil && resp.Results[0].Status != ua.StatusOK {
		s.logger.Errorf("Status not OK: %v", resp.Results[0].Status)
	}
	return resp, nil
}

// Listen for data changes
func (s *opcSensor) monitorData() error {
	// read from subscription's notification channel until ctx is cancelled
	s.logger.Infof("Go routine started")
	for {
		select {
		case <-s.cancelCtx.Done():
			s.logger.Infof("Go routine ended")
			return nil
		case res := <-s.notifyChannel:
			if res.Error != nil {
				s.logger.Error(res.Error)
				continue
			}
			switch x := res.Value.(type) {
			case *ua.DataChangeNotification:
				for _, item := range x.MonitoredItems {
					s.queue.PushBack(item.Value.Value.Value())
					s.queueEmpty = false
					s.logger.Infof("MonitoredItem with client handle %v = %v", item.ClientHandle, s.queue.Back())
				}
			}
		}
	}
}

// Creates monitored item request from node ids
func valueRequest(nodeID *ua.NodeID) *ua.MonitoredItemCreateRequest {
	handle := uint32(42)
	return opcua.NewMonitoredItemCreateRequestWithDefaults(nodeID, ua.AttributeIDValue, handle)
}
