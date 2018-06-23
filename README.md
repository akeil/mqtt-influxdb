# MQTT-InfluxDB
Subscribe to [MQTT](https://mqtt.org/) topics
and submit messages as measurements to [InfluxDB](https://www.influxdata.com/time-series-platform/influxdb/).


## Configuration
Configuration files are stored at

- `~/.config/mqtt-influxdb.json`
- `/etc/mqtt-influxdb.json`

The user specific file takes precedence over the system-wide configuration.

The path to the configuration file can be specified on the command line:

```sh
$ mfx -c /path/to/config.json
```

If the configuration is specified on the command line, it *must* exist.
Otherwise, a default configuration will be used if no file can be found.

The configuration file looks like this:

```json
{
    "pidfile": "/tmp/mfx.pid",
    "MQTTHost": "localhost",
    "MQTTPort": 1883,
    "MQTTUser": "username",
    "MQTTPass": "secret",
    "influxHost": "localhost",
    "influxPort": 8086,
    "influxUser": "username",
    "influxPass": "secret",
    "influxDB": "default"
}
```
Configuration keys and default values:

| Key        | Default   | Description                                       |
|------------|-----------|---------------------------------------------------|
| pidfile    | *empty*   | If set to a path, write PID file to that location |
| MQTTHost   | localhost | Hostname or IP address for MQTT broker            |
| MQTTPort   | 1883      | Port for MQTT broker                              |
| MQTTUser   | *empty*   | Username for MQTT authentication                  |
| MQTTPass   | *empty*   | Password for MQTT authentication                  |
| influxHost | localhost | Hostname or IP address of InfluxDB                |
| influxPort | 8086      | Port for InfluxDB                                 |
| influxUser | *empty*   | Username for authenticating against InfluxDB      |
| influxPass | *empty*   | Password (clear) for InfluxDB                     |
| influxDB   | default   | Name of the InfluxDB database                     |


## Subscriptions
Keep several JSON files in the subscription directory:

- `/etc/mqtt-influxdb.d`
- `~/.config/mqtt-influxdb.d`

Each file should contain an array with subscription details.
A file with a single Subscription might look like this:

```json
[
  {
    "topic": "home/+/thermostat/status/actual_temperature",
    "measurement": "temperature",
    "tags": {
      "device": "thermostat",
      "room": "{{.Part 1}}"
    },
    "conversion": {
      "kind": "float",
      "precision": 1
    }
  }
]
```

Each subscription defines a single MQTT *topic* (possibly using wildcards)
to subscribe to and an InfluxDB *measurement* to submit values to.
Optionally a *conversion* can be specified.

| Key                   | Description                           |
|-----------------------|---------------------------------------|
| `topic`               | The MQTT topic to subscribe to        |
| `measurement`         | The name of the InfluxDB measurement  |
| `tags`                | A map with tag names and their values |
| `tags.[TAG]`          | a tag name and the tag value          |
| `conversion`          | Conversion details                    |
| `conversion.kind`     | The type of conversion to apply       |
| `conversion.[OPTION]` | Conversion options, depends on `kind` |


### Dynamic Values for Measurements or Tags
The values for the measurement and tags can be determined dynamically from the
MQTT topic. This is useful if the topic contains wildcards.
To get the *nth* element from the topic path, use the `{{.Part n}}` template.
The path index is zero-based.
Examples:

| Template                  | Topic       | Result    |
|---------------------------|-------------|-----------|
| `{{.Part 1}} `            | foo/bar/baz | "bar"     |
| `{{.Part 1}}-{{.Part 0}}` | foo/bar/baz | "bar-foo" |


## Conversions
By default, the MQTT message is treated as a string value.

When submitting to InfluxDB, values are also submitted as strings,
but InfluxDB will determine the data type based on the format of the value.
Moreover, the data type for an influx measurement is decided by the first value
that is submitted. This can cause problems when values should be *floats*
and (only) the first value is submitted as "1" instead of "1.0".


### Float
Values are converted to floating point numbers and rounded to the given
**precision** (number of decimal places).

If a value for **scale** is defined, the value will be multiplied by that value
(before rounding). The scale value is a float.


### Integer
Convert to integer with optional **scale** (same as for float).


### String
Treats the value as a string.


### Boolean
Converts to a boolean value, accepts the same formats the Go
[ParseBool](https://golang.org/pkg/strconv/#ParseBool) function:

- *true*: 1, t, T, TRUE, true or True
- *false*: 0, f, F, FALSE, false or False


### On-Off
Expects MQTT messages to contain either "on" or "off" (case-insensitive)
and converts to a boolean value with `on=true`.
