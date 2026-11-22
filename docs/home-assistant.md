# Home Assistant read-only export

EnerFlux owns the measurements, calculation rules, and prices. Home Assistant reads a small set of calculated values from one read-only endpoint; it does not calculate pellet, water, electricity, or cost estimates.

## Configure the token in EnerFlux

Generate a long random token on the machine used to manage the deployment, then add it to EnerFlux's `.env` file:

```sh
openssl rand -hex 32
```

```dotenv
HOME_ASSISTANT_READ_TOKEN=<generated-value>
```

Deploy EnerFlux again so the value is loaded into the API container's Kubernetes Secret. Keep the token private. Rotate it by generating a new value, updating `.env`, and redeploying. An empty setting disables the export endpoint.

## Configure Home Assistant

Add the same token to Home Assistant's `secrets.yaml`:

```yaml
enerflux_read_token: "Bearer <generated-value>"
```

Then add REST sensors to `configuration.yaml`, replacing the host and port with the EnerFlux API address:

```yaml
rest:
  - resource: http://<enerflux-host>:30080/api/v1/integrations/home-assistant/sensors
    scan_interval: 300
    headers:
      Authorization: !secret enerflux_read_token
    sensor:
      - name: EnerFlux water meter total
        unique_id: enerflux_water_meter_total
        value_template: "{{ value_json.water_meter_total_m3 | float(none) }}"
        unit_of_measurement: m³
      - name: EnerFlux pellet stock
        unique_id: enerflux_pellet_stock
        value_template: "{{ value_json.pellet_stock_kg | float(none) }}"
        unit_of_measurement: kg
      - name: EnerFlux pellets consumed last 30 days
        unique_id: enerflux_pellets_consumed_30_days
        value_template: "{{ value_json.pellets_consumed_last_30_days_kg | float(none) }}"
        unit_of_measurement: kg
      - name: EnerFlux average pellet use per covered day
        unique_id: enerflux_pellet_average_per_covered_day_30_days
        value_template: "{{ value_json.pellet_average_per_covered_day_30_days_kg | float(none) }}"
        unit_of_measurement: kg/d
      - name: EnerFlux pellet forecast next hour
        unique_id: enerflux_pellet_forecast_next_hour
        value_template: "{{ value_json.pellet_forecast_next_hour_kg | float(none) }}"
        unit_of_measurement: kg
      - name: EnerFlux water consumed last 30 days
        unique_id: enerflux_water_consumed_30_days
        value_template: "{{ value_json.water_consumed_last_30_days_m3 | float(none) }}"
        unit_of_measurement: m³
      - name: EnerFlux electricity consumed today
        unique_id: enerflux_electricity_consumed_today
        value_template: "{{ value_json.electricity_consumed_today_kwh | float(none) }}"
        unit_of_measurement: kWh
      - name: EnerFlux electricity cost last 30 days
        unique_id: enerflux_electricity_cost_30_days
        value_template: "{{ value_json.electricity_cost_last_30_days_eur | float(none) }}"
        unit_of_measurement: EUR
      - name: EnerFlux water cost last 30 days
        unique_id: enerflux_water_cost_30_days
        value_template: "{{ value_json.water_cost_last_30_days_eur | float(none) }}"
        unit_of_measurement: EUR
      - name: EnerFlux pellet cost last 30 days
        unique_id: enerflux_pellet_cost_30_days
        value_template: "{{ value_json.pellet_cost_last_30_days_eur | float(none) }}"
        unit_of_measurement: EUR
      - name: EnerFlux total resource cost last 30 days
        unique_id: enerflux_total_cost_30_days
        value_template: "{{ value_json.total_cost_last_30_days_eur | float(none) }}"
        unit_of_measurement: EUR
```

Values can be `null` until the relevant readings or prices exist. The combined 30-day cost is null unless all three domain costs can be calculated for the same period. EnerFlux returns `generated_at` and its configured `time_zone` with the sensor values. The endpoint exposes water meter total, water consumption and cost, electricity consumption and cost, pellet stock, pellet use and cost, average daily pellet use, the next-hour estimate, and a complete combined cost when available. See [Home Assistant REST sensor configuration](https://www.home-assistant.io/integrations/sensor.rest/) for REST sensor options.

Use HTTPS when the request crosses an untrusted network. The endpoint accepts only `GET` and requires the configured bearer token. Other EnerFlux APIs retain their existing access behavior.
