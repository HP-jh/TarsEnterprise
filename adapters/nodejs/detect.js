#!/usr/bin/env node
/* TarsEnterprise Node.js Adapter — Environment Detection & Tool Listing */
const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');

function detect() {
    const result = {
        lang: 'nodejs',
        version: process.version,
        interpreter: process.execPath,
        packages: [],
        tools: []
    };
    // List global packages
    try {
        const globalRoot = execSync('npm root -g', { encoding: 'utf8' }).trim();
        if (fs.existsSync(globalRoot)) {
            result.packages = fs.readdirSync(globalRoot).filter(d => !d.startsWith('.'));
        }
    } catch (e) {}
    // Check common tools
    const tools = ['npm', 'yarn', 'pnpm', 'eslint', 'prettier', 'typescript', 'ts-node'];
    for (const tool of tools) {
        try {
            execSync(`${tool} --version`, { encoding: 'utf8', stdio: 'pipe' });
            result.tools.push(tool);
        } catch (e) {}
    }
    return result;
}

console.log(JSON.stringify(detect(), null, 2));
