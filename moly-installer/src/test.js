#!/usr/bin/env node

import chalk from 'chalk';
import { SystemRequirements } from './systemRequirements.js';
import { DownloadManager } from './downloadManager.js';
import fetch from 'node-fetch';

async function testSystemRequirements() {
  console.log(chalk.bold.cyan('\n📋 Test 1: System Requirements Checker\n'));

  const sysReq = new SystemRequirements();
  await sysReq.check();

  console.log(chalk.dim('OS:'), sysReq.requirements.os?.name);
  console.log(chalk.dim('RAM:'), sysReq.requirements.ram?.size);
  console.log(chalk.dim('Disk:'), sysReq.requirements.disk?.size);
  console.log(chalk.dim('CPU:'), `${sysReq.requirements.cpu?.cores} cores`);

  const results = sysReq.getResults();
  if (results.critical.length > 0) {
    console.log(chalk.red('Critical issues found:'), results.critical);
    return false;
  }

  console.log(chalk.green('✓ System requirements check passed'));
  return true;
}

async function testDownloadURLs() {
  console.log(chalk.bold.cyan('\n🌐 Test 2: Download URLs Availability\n'));

  const urls = {
    'Ollama (macOS)': 'https://ollama.ai/download/Ollama-darwin.zip',
    'Ollama (Windows)': 'https://ollama.ai/download/OllamaSetup.exe',
    'Ollama (Linux)': 'https://ollama.ai/download/ollama-linux-x86_64.tar.gz',
    'LM Studio (macOS)': 'https://lmstudio.ai/api/download/mac',
    'LM Studio (Windows)': 'https://lmstudio.ai/api/download/windows',
    'LM Studio (Linux)': 'https://lmstudio.ai/api/download/linux',
  };

  let passed = 0;
  let failed = 0;

  for (const [name, url] of Object.entries(urls)) {
    try {
      const response = await fetch(url, {
        method: 'HEAD',
        redirect: 'follow',
        timeout: 5000,
      });

      if (response.ok || response.status === 302) {
        console.log(chalk.green(`✓ ${name}`));
        passed++;
      } else {
        console.log(chalk.yellow(`⚠ ${name}: ${response.status}`));
        failed++;
      }
    } catch (error) {
      console.log(chalk.red(`✗ ${name}: ${error.message}`));
      failed++;
    }
  }

  console.log(chalk.dim(`\nResults: ${passed} available, ${failed} unreachable`));

  if (failed > 0) {
    console.log(chalk.yellow('Note: Some URLs may be down or behind redirects'));
  }

  return failed === 0;
}

async function testOllamaConnectivity() {
  console.log(chalk.bold.cyan('\n🦙 Test 3: Ollama Connectivity\n'));

  try {
    const response = await fetch('http://localhost:11434/api/tags', {
      signal: AbortSignal.timeout(3000),
    });

    if (response.ok) {
      const data = await response.json();
      console.log(chalk.green('✓ Ollama is running and responding'));
      console.log(chalk.dim(`Available models: ${data.models?.length || 0}`));

      if (data.models && data.models.length > 0) {
        console.log(chalk.dim('Models:', data.models.map(m => m.name).join(', ')));
      }

      return true;
    } else {
      console.log(chalk.yellow('⚠ Ollama responded but with error status:', response.status));
      return false;
    }
  } catch (error) {
    console.log(chalk.yellow('ℹ Ollama is not running (this is normal if not installed)'));
    console.log(chalk.dim('Start with: ollama serve'));
    return false;
  }
}

async function testCORSProxyConnectivity() {
  console.log(chalk.bold.cyan('\n🔄 Test 4: CORS Proxy Connectivity\n'));

  try {
    const response = await fetch('http://localhost:11435/api/tags', {
      signal: AbortSignal.timeout(3000),
    });

    if (response.ok) {
      const corsHeader = response.headers.get('access-control-allow-origin');
      console.log(chalk.green('✓ CORS Proxy is running and responding'));
      console.log(chalk.dim(`CORS header: ${corsHeader || 'not found'}`));
      return true;
    } else {
      console.log(chalk.yellow('⚠ Proxy responded with error:', response.status));
      return false;
    }
  } catch (error) {
    console.log(chalk.yellow('ℹ CORS Proxy is not running (this is normal if not installed)'));
    console.log(chalk.dim('Start with: moly-proxy'));
    return false;
  }
}

async function testNodeJS() {
  console.log(chalk.bold.cyan('\n⚙️ Test 5: Node.js Environment\n'));

  console.log(chalk.dim('Node version:'), process.version);
  console.log(chalk.dim('NPM version:'), process.env.npm_version || 'unknown');
  console.log(chalk.dim('Platform:'), process.platform);
  console.log(chalk.dim('Architecture:'), process.arch);

  const [major] = process.version.slice(1).split('.');
  if (parseInt(major) >= 16) {
    console.log(chalk.green('✓ Node.js version >= 16'));
    return true;
  } else {
    console.log(chalk.red('✗ Node.js 16+ required'));
    return false;
  }
}

async function testDependencies() {
  console.log(chalk.bold.cyan('\n📦 Test 6: Required Dependencies\n'));

  const deps = ['chalk', 'prompts', 'cli-progress'];
  let allInstalled = true;

  for (const dep of deps) {
    try {
      await import(dep);
      console.log(chalk.green(`✓ ${dep}`));
    } catch {
      console.log(chalk.red(`✗ ${dep} not found`));
      allInstalled = false;
    }
  }

  if (!allInstalled) {
    console.log(chalk.yellow('\nInstall missing dependencies with: npm install'));
  }

  return allInstalled;
}

async function testJSON() {
  console.log(chalk.bold.cyan('\n📄 Test 7: Configuration Files\n'));

  try {
    const { readFileSync } = await import('fs');
    const packageJson = JSON.parse(
      readFileSync(new URL('../package.json', import.meta.url), 'utf-8')
    );

    console.log(chalk.green('✓ package.json is valid JSON'));
    console.log(chalk.dim(`Name: ${packageJson.name}`));
    console.log(chalk.dim(`Version: ${packageJson.version}`));

    return true;
  } catch (error) {
    console.log(chalk.red(`✗ Invalid package.json: ${error.message}`));
    return false;
  }
}

async function runAllTests() {
  console.log(chalk.bold.cyan(`
╔════════════════════════════════════════╗
║    Moly Installer - Test Suite         ║
╚════════════════════════════════════════╝
`));

  const results = {
    systemRequirements: await testSystemRequirements(),
    nodeJS: await testNodeJS(),
    dependencies: await testDependencies(),
    configFiles: await testJSON(),
    downloadURLs: await testDownloadURLs(),
    ollamaConnectivity: await testOllamaConnectivity(),
    corsProxyConnectivity: await testCORSProxyConnectivity(),
  };

  // Summary
  console.log(chalk.bold.cyan('\n📊 Test Summary\n'));

  const tests = Object.entries(results);
  const passed = tests.filter(([, result]) => result).length;
  const total = tests.length;

  for (const [name, result] of tests) {
    const icon = result ? chalk.green('✓') : chalk.yellow('⚠');
    console.log(`${icon} ${name}`);
  }

  console.log(chalk.dim(`\nPassed: ${passed}/${total}`));

  if (passed === total) {
    console.log(chalk.green.bold('\n✓ All tests passed!'));
    console.log(chalk.green.bold('Ready to run: moly-installer\n'));
  } else {
    console.log(chalk.yellow.bold('\n⚠ Some tests failed or inconclusive'));
    console.log(chalk.yellow('This may be normal if components are not installed yet\n'));
  }
}

runAllTests().catch(error => {
  console.error(chalk.red('Test suite error:', error));
  process.exit(1);
});
