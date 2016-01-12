package cli

import (
	"errors"
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/bitrise-tools/bitrise-machine/config"
	"github.com/bitrise-tools/bitrise-machine/utils"
	"github.com/bitrise-tools/bitrise-machine/vagrant"
	"github.com/codegangsta/cli"
)

func getVagrantStatus(configModel config.MachineConfigModel) (vagrant.MachineReadableItem, error) {
	// Read `vagrant status` log/output
	outputs, err := utils.RunAndReturnCombinedOutput(MachineWorkdir.Get(), configModel.Envs.ToCmdEnvs(), "vagrant", "status", "--machine-readable")
	if err != nil {
		return vagrant.MachineReadableItem{}, fmt.Errorf("'vagrant status' failed. Output was: %s", outputs)
	}
	statusItms := vagrant.ParseMachineReadableItemsFromString(outputs, "", "state")
	if len(statusItms) != 1 {
		return vagrant.MachineReadableItem{}, fmt.Errorf("Failed to determine the 'status' of the machine. Output was: %s", outputs)
	}
	return statusItms[0], nil
}

// destroyCommon ...
//  common code, cleanup's destroy
func destroyCommon(configModel config.MachineConfigModel) error {
	machineStatus, err := getVagrantStatus(configModel)
	if err != nil {
		return fmt.Errorf("Failed to get vagrant status: %s", err)
	}

	if machineStatus.Data != "not_created" {
		// destroy
		log.Infoln("Destroying machine...")
		if err := doDestroy(configModel); err != nil {
			return fmt.Errorf("'vagrant destroy' failed with error: %s", err)
		}
		log.Infoln("Machine destroyed.")
	} else {
		log.Infoln("Machine is in not-created state, skipping destroy.")
	}

	return nil
}

func doRecreateCleanup(configModel config.MachineConfigModel) error {
	// destroy
	if err := destroyCommon(configModel); err != nil {
		return fmt.Errorf("doRecreateCleanup: failed to destroy: %s", err)
	}

	// re-create
	if err := utils.Run(MachineWorkdir.Get(), configModel.Envs.ToCmdEnvs(), "vagrant", "up"); err != nil {
		return fmt.Errorf("'vagrant up' failed with error: %s", err)
	}

	log.Infoln("Machine created and ready!")
	return nil
}

func doDestroyCleanup(configModel config.MachineConfigModel) error {
	// destroy
	if err := destroyCommon(configModel); err != nil {
		return fmt.Errorf("doRecreateCleanup: failed to destroy: %s", err)
	}

	log.Infoln("Machine destroyed, clean!")
	return nil
}

func doCustomCleanup(configModel config.MachineConfigModel) error {
	log.Infoln("Cleanup mode: custom-command")
	if configModel.CustomCleanupCommand == "" {
		return errors.New("Cleanup mode was custom-command, but no custom cleanup command specified!")
	}
	log.Infof("=> Specified custom command: %s", configModel.CustomCleanupCommand)

	// Read `vagrant status` log/output
	machineStatus := vagrant.MachineReadableItem{}
	if outputs, err := utils.RunAndReturnCombinedOutput(MachineWorkdir.Get(), configModel.Envs.ToCmdEnvs(), "vagrant", "status", "--machine-readable"); err != nil {
		if err != nil {
			log.Errorf("'vagrant status' failed with output: %s", outputs)
			return err
		}
	} else {
		statusItms := vagrant.ParseMachineReadableItemsFromString(outputs, "", "state")
		if len(statusItms) != 1 {
			return fmt.Errorf("Failed to determine the 'status' of the machine. Output was: %s", outputs)
		}
		machineStatus = statusItms[0]
	}

	if machineStatus.Data == "not_created" {
		log.Infoln("Machine not yet created - creating with 'vagrant up'...")
		if err := utils.Run(MachineWorkdir.Get(), configModel.Envs.ToCmdEnvs(), "vagrant", "up"); err != nil {
			return fmt.Errorf("'vagrant up' failed with error: %s", err)
		}
		log.Infoln("Machine created!")
	} else {
		log.Infof("Machine already created - using the specified custom-command (%s) to clean it up...", configModel.CustomCleanupCommand)
		if err := utils.Run(MachineWorkdir.Get(), configModel.Envs.ToCmdEnvs(), "vagrant", configModel.CustomCleanupCommand); err != nil {
			return fmt.Errorf("'vagrant %s' failed with error: %s", configModel.CustomCleanupCommand, err)
		}
		log.Infoln("Successful custom cleanup")
	}

	log.Infoln("Machine created and ready!")
	return nil
}

// doCleanup ...
// @isSkipHostCleanup : !!! should only be specified in case the host will be destroyed right after
//   the cleanup. 'will-be-destroyed' will leave the host as-it-is, uncleared!!
func doCleanup(configModel config.MachineConfigModel, isSkipHostCleanup string) error {
	log.Infof("==> doCleanup (mode: %s)", configModel.CleanupMode)

	if isSkipHostCleanup != "will-be-destroyed" {
		if configModel.CleanupMode == config.CleanupModeRollback {
			if err := utils.Run(MachineWorkdir.Get(), configModel.Envs.ToCmdEnvs(), "vagrant", "sandbox", "rollback"); err != nil {
				return err
			}
		} else if configModel.CleanupMode == config.CleanupModeRecreate {
			if err := doRecreateCleanup(configModel); err != nil {
				return err
			}
		} else if configModel.CleanupMode == config.CleanupModeDestroy {
			if err := doDestroyCleanup(configModel); err != nil {
				return err
			}
		} else if configModel.CleanupMode == config.CleanupModeCustomCommand {
			if err := doCustomCleanup(configModel); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("Unsupported CleanupMode: %s", configModel.CleanupMode)
		}
	} else {
		log.Warnln("Skipping Host Cleanup! This option should only be used if the Host is destroyed immediately after this cleanup!!")
	}

	if err := config.DeleteSSHFilesFromDir(MachineWorkdir.Get()); err != nil {
		return fmt.Errorf("Failed to delete SSH file from workdir: %s", err)
	}

	return nil
}

func cleanup(c *cli.Context) {
	log.Infoln("Cleanup")

	additionalEnvs, err := config.CreateEnvItemsModelFromSlice(MachineParamsAdditionalEnvs.Get())
	if err != nil {
		log.Fatalf("Invalid Environment parameter: %s", err)
	}
	log.Debugf("additionalEnvs: %#v", additionalEnvs)

	configModel, err := config.ReadMachineConfigFileFromDir(MachineWorkdir.Get(), additionalEnvs)
	if err != nil {
		log.Fatalln("Failed to read Config file: ", err)
	}

	log.Infof("configModel: %#v", configModel)

	if err := doCleanup(configModel, ""); err != nil {
		log.Fatalf("Failed to Cleanup: %s", err)
	}

	log.Infoln("Cleanup - DONE - OK")
}
