## Plugin

Plugin is an important feature of Subscan Essentials.
It is very convenient to save **Extrinsic** and **Event** to the database and parse it into various customized content, save it in a new data table, and display the customization data to the frontend through HTTP API.

### Usage

1. Refer [plugin](https://github.com/itering/subscan-plugin) to write the plugin you need

1. Just import your plugin in ``plugins/registry.go`` like

```
func init() {
	registerNative(YourPlugin.New()) // Register plugin to subscan
}
```

"block_num","block_timestamp","hash","parent_hash","state_root","extrinsics_root","logs","extrinsics","event_count","extrinsics_count","event","spec_version","validator","codec_error","finalized"

LOAD DATA LOCAL INFILE '/tmp/chain_blocks_5_202110110929.csv' INTO TABLE chain_blocks_5 (block_num, block_timestamp, hash, parent_hash, state_root, extrinsics_root, logs, extrinsics, event_count, extrinsics_count, event, spec_version, validator, codec_error, finalized) FIELDS TERMINATED BY ',' ENCLOSED BY '"' LINES TERMINATED BY '\r\n' IGNORE 1 LINES;