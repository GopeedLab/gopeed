#!/usr/bin/env node

/**
 * Gopeed Script Example: Process downloaded file
 * This script demonstrates how to process downloaded files
 * (e.g., extract archives, convert formats, etc.)
 */

const fs = require('fs');
const path = require('path');
const { exec } = require('child_process');

// Get environment variables provided by Gopeed
const event = process.env.GOPEED_EVENT;
const taskName = process.env.GOPEED_TASK_NAME;
const filePath = process.env.GOPEED_FILE_PATH;
const fileName = process.env.GOPEED_FILE_NAME;

// Only process DOWNLOAD_DONE events
if (event !== 'DOWNLOAD_DONE') {
    console.log('Event is not DOWNLOAD_DONE, skipping');
    process.exit(0);
}

console.log(`Processing downloaded file: ${taskName}`);
console.log(`File path: ${filePath}`);

// Check if file exists
if (!fs.existsSync(filePath)) {
    console.error(`Error: File not found at ${filePath}`);
    process.exit(1);
}

// Example: Extract zip files automatically
if (fileName.endsWith('.zip')) {
    const extractDir = path.join(path.dirname(filePath), path.basename(fileName, '.zip'));
    
    console.log(`Extracting ${fileName} to ${extractDir}`);
    
    exec(`unzip -q "${filePath}" -d "${extractDir}"`, (error, stdout, stderr) => {
        if (error) {
            console.error(`Error extracting file: ${error.message}`);
            process.exit(1);
        }
        
        if (stderr) {
            console.error(`stderr: ${stderr}`);
        }
        
        console.log('File extracted successfully');
        console.log(stdout);
        
        // Optionally delete the zip file after extraction
        // fs.unlinkSync(filePath);
        // console.log('Original zip file deleted');
    });
} else {
    console.log('File is not a zip archive, no processing needed');
}

// You can also read the full task data from stdin
let stdinData = '';
process.stdin.on('data', chunk => {
    stdinData += chunk;
});

process.stdin.on('end', () => {
    if (stdinData) {
        try {
            const taskData = JSON.parse(stdinData);
            console.log(`Full task data available: ${taskData.event}`);
        } catch (e) {
            // Ignore parsing errors
        }
    }
});
