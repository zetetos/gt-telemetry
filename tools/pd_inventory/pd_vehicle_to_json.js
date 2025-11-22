#!/usr/bin/env node

const fs = require('fs');
const path = require('path');

// Import the data
const { Cars } = require('./pd_inventory.js');

// Function to convert object to JSON with reformatted IDs
function convertToJSON(data) {
    const result = {};

    // Process each car
    for (const [key, carData] of Object.entries(data)) {
        // Extract numeric ID from car1234 format
        const numericId = key.replace(/^car/, '');

        // Create new object with all original data
        result[numericId] = carData;
    }

    return JSON.stringify(result, null, 2);
}

// Main execution
try {
    console.log('Converting pd_inventory.js to JSON...');

    const jsonContent = convertToJSON(Cars);
    const outputPath = path.join(__dirname, 'pd_inventory.json');

    fs.writeFileSync(outputPath, jsonContent, 'utf8');

    console.log(`✓ Successfully converted to ${outputPath}`);
    console.log(`  Total records: ${Object.keys(Cars).length}`);
} catch (error) {
    console.error('Error converting to JSON:', error.message);
    process.exit(1);
}
