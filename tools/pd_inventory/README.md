# Polyphony Digital Inventory

Polyphony Digital provide an online list of all current vehicles along with their key characteristics via the Gran Turismo 7 website.


## Merging the vehicle inventory

1. Visit https://www.gran-turismo.com/gb/gt7/carlist and check the sources that are downloaded to present the page.
2. Locate the source named `cars.gb-<hash>.js` and save the contents to a file named `pd_inventory.js`. _The hash will probably change whenever a new version of Gran Turismo is released and new vehicles are added._
3. Convert the JavaScript formatted data to JSON:

   `node pd_vehicle_to_json.js`
4. Merge the GT7 data into the GT Telemetry data:

   `go run cmd/vehicle_inventory/main.go merge pkg/vehicles/vehicles.json pd_inventory.json > pkg/vehicles/vehicles.json`
5. Commit the changes when satisfied with the results.