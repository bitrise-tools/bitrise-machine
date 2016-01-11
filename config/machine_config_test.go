package config

import (
	"testing"

	"github.com/bitrise-io/go-utils/testutil"
	"github.com/stretchr/testify/require"
)

func Test_EnvItemsModel_ToCmdEnvs(t *testing.T) {
	empty := EnvItemsModel{}
	require.Equal(t, []string{}, empty.ToCmdEnvs())

	one := EnvItemsModel{"key": "value"}
	require.Equal(t, []string{"key=value"}, one.ToCmdEnvs())

	two := EnvItemsModel{"key1": "value 1", "key2": "value 2"}
	testutil.EqualSlicesWithoutOrder(t, []string{"key1=value 1", "key2=value 2"}, two.ToCmdEnvs())

	envRef := EnvItemsModel{"key": "value with $HOME env ref"}
	require.Equal(t, []string{"key=value with $HOME env ref"}, envRef.ToCmdEnvs())
}

func Test_readMachineConfigFromBytes(t *testing.T) {
	configContent := `{
"cleanup_mode": "rollback",
"is_cleanup_before_setup": false,
"is_do_timesync_at_setup": true
}`

	t.Log("configContent: ", configContent)

	t.Log("Base Config")
	{
		configModel, err := readMachineConfigFromBytes([]byte(configContent), EnvItemsModel{})
		require.NoError(t, err)
		t.Logf("configModel: %#v", configModel)

		if configModel.CleanupMode != "rollback" {
			t.Fatal("Invalid CleanupMode!")
		}
		if configModel.IsCleanupBeforeSetup != false {
			t.Fatal("Invalid IsCleanupBeforeSetup!")
		}

		require.Equal(t, []string{}, configModel.Envs.ToCmdEnvs())
	}

	t.Log("Additional Env items")
	{
		configModel, err := readMachineConfigFromBytes([]byte(configContent), EnvItemsModel{"key": "my value"})
		require.NoError(t, err)
		t.Logf("configModel: %#v", configModel)
		require.Equal(t, EnvItemsModel{"key": "my value"}, configModel.Envs)
	}

	t.Log("Additional Env items - overwrite a config defined Env")
	{
		configContent := `{
"cleanup_mode": "rollback",
"is_cleanup_before_setup": false,
"is_do_timesync_at_setup": true,
"envs": {
  "MY_KEY": "config value"
}
}`
		configModel, err := readMachineConfigFromBytes([]byte(configContent), EnvItemsModel{"MY_KEY": "additional env value"})
		require.NoError(t, err)
		t.Logf("configModel: %#v", configModel)
		require.Equal(t, EnvItemsModel{"MY_KEY": "additional env value"}, configModel.Envs)
	}
}

func Test_MachineConfigModel_normalizeAndValidate(t *testing.T) {
	configModel := MachineConfigModel{CleanupMode: ""}
	t.Log("Invalid CleanupMode")
	if err := configModel.normalizeAndValidate(); err == nil {
		t.Fatal("Should return a validation error!")
	}

	configModel = MachineConfigModel{
		CleanupMode:          CleanupModeRollback,
		IsCleanupBeforeSetup: true,
		IsDoTimesyncAtSetup:  false,
	}

	t.Logf("configModel: %#v", configModel)
	if err := configModel.normalizeAndValidate(); err != nil {
		t.Fatalf("Failed with error: %s", err)
	}
	if configModel.IsCleanupBeforeSetup != true {
		t.Fatal("Invalid IsCleanupBeforeSetup")
	}
	if configModel.IsDoTimesyncAtSetup != false {
		t.Fatal("Invalid IsDoTimesyncAtSetup")
	}
}

func TestCreateEnvItemsModelFromSlice(t *testing.T) {
	t.Log("Empty")
	{
		envsItmModel, err := CreateEnvItemsModelFromSlice([]string{})
		require.NoError(t, err)
		require.Equal(t, EnvItemsModel{}, envsItmModel)
	}

	t.Log("One item - but invalid, empty")
	{
		_, err := CreateEnvItemsModelFromSlice([]string{""})
		require.EqualError(t, err, "Invalid item, empty key. (Parameter was: )")
	}

	t.Log("One item - but invalid, value provided but empty key")
	{
		_, err := CreateEnvItemsModelFromSlice([]string{"=hello"})
		require.EqualError(t, err, "Invalid item, empty key. (Parameter was: =hello)")
	}

	t.Log("One item, no value - error")
	{
		_, err := CreateEnvItemsModelFromSlice([]string{"a"})
		require.EqualError(t, err, "Invalid item, no value defined. Key was: a")
	}

	t.Log("One item, with value")
	{
		envsItmModel, err := CreateEnvItemsModelFromSlice([]string{"a=b"})
		require.NoError(t, err)
		require.Equal(t, EnvItemsModel{"a": "b"}, envsItmModel)
	}

	t.Log("One item, with empty value")
	{
		envsItmModel, err := CreateEnvItemsModelFromSlice([]string{"a="})
		require.NoError(t, err)
		require.Equal(t, EnvItemsModel{"a": ""}, envsItmModel)
	}

	t.Log("One item, with value which includes spaces")
	{
		envsItmModel, err := CreateEnvItemsModelFromSlice([]string{"a=b c  d"})
		require.NoError(t, err)
		require.Equal(t, EnvItemsModel{"a": "b c  d"}, envsItmModel)
	}

	t.Log("One item, with value which includes equal signs")
	{
		envsItmModel, err := CreateEnvItemsModelFromSlice([]string{"a=b c=d  =e"})
		require.NoError(t, err)
		require.Equal(t, EnvItemsModel{"a": "b c=d  =e"}, envsItmModel)
	}

	t.Log("Multiple values")
	{
		envsItmModel, err := CreateEnvItemsModelFromSlice([]string{"a=b c d", "1=2 3 4"})
		require.NoError(t, err)
		require.Equal(t, EnvItemsModel{"a": "b c d", "1": "2 3 4"}, envsItmModel)
	}
}
