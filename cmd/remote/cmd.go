package main

import (
	"context"
	"os"

	"encoding/json"

	"go.viam.com/rdk/module"

	"go.viam.com/rdk/components/sensor"
	"go.viam.com/rdk/config"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	robotimpl "go.viam.com/rdk/robot/impl"
	"go.viam.com/rdk/robot/web"
	_ "go.viam.com/rdk/services/sensors/builtin"
	rdkutils "go.viam.com/rdk/utils"
	"go.viam.com/utils"

	"github.com/felixreichenbach/opc-ua/opcsensor"
)

func main() {
	// NewLoggerFromArgs will create a logging.Logger at "DebugLevel" if
	// "--log-level=debug" is the 3rd argument in os.Args and at "InfoLevel" otherwise.
	utils.ContextualMain(mainWithArgs, module.NewLoggerFromArgs("testsensor"))
}

func mainWithArgs(ctx context.Context, args []string, logger logging.Logger) (err error) {

	logger.Infof("os.ARgs: %v", os.Args)

	netconfig := config.NetworkConfig{}
	netconfig.BindAddress = "0.0.0.0:8083"

	if err := netconfig.Validate(""); err != nil {
		return err
	}

	var nodeIDs []string
	json.Unmarshal([]byte(os.Args[4]), &nodeIDs)

	// Update the Attributes and ConvertedAttributes with the attributes your modular resource should receive
	conf := &config.Config{
		Network: netconfig,
		Components: []resource.Config{
			{
				Name:  os.Args[1],
				API:   sensor.API,
				Model: opcsensor.Model,
				Attributes: rdkutils.AttributeMap{
					"endpoint":  os.Args[3],
					"nodeids":   nodeIDs,
					"subscribe": "data",
				},
				ConvertedAttributes: &opcsensor.Config{
					Endpoint:  os.Args[3],
					NodeIDs:   nodeIDs,
					Subscribe: "data",
				},
			},
		},
	}

	myRobot, err := robotimpl.New(ctx, conf, logger)
	if err != nil {
		return err
	}

	return web.RunWebWithConfig(ctx, myRobot, conf, logger)
}
