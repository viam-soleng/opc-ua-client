// Package main is a module which serves the customsensor custom model.
package main

import (
	"context"

	"github.com/felixreichenbach/opc-ua/opcsensor"
	"go.viam.com/rdk/components/sensor"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/module"
	"go.viam.com/utils"
)

func main() {
	// NewLoggerFromArgs will create a logging.Logger at "DebugLevel" if
	// "--log-level=debug" the 3rd argument in os.Args and at "InfoLevel" otherwise.
	utils.ContextualMain(mainWithArgs, module.NewLoggerFromArgs("opcsensor"))
}

func mainWithArgs(ctx context.Context, args []string, logger logging.Logger) (err error) {
	myModule, err := module.NewModuleFromArgs(ctx, logger)
	if err != nil {
		return err
	}

	// Adds the preregistered sensor component API to the module for the new model
	err = myModule.AddModelFromRegistry(ctx, sensor.API, opcsensor.Model)
	if err != nil {
		return err
	}

	err = myModule.Start(ctx)
	defer myModule.Close(ctx)
	if err != nil {
		return err
	}
	<-ctx.Done()
	return nil
}
