#!/usr/bin/env node

/**
 * Pre-Submission Verification Script
 * Checks all requirements are met before submitting to Chrome Web Store
 */

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const projectRoot = path.join(__dirname, '..');

let errors = [];
let warnings = [];
let success = [];

function checkFile(filePath, description) {
  const fullPath = path.join(projectRoot, filePath);
  if (fs.existsSync(fullPath)) {
    success.push(`✓ ${description}`);
    return true;
  } else {
    errors.push(`✗ Missing: ${description} (${filePath})`);
    return false;
  }
}

function checkManifest() {
  try {
    const manifest = JSON.parse(fs.readFileSync(path.join(projectRoot, 'manifest.json'), 'utf8'));

    const checks = [
      { key: 'manifest_version', value: 3, desc: 'Manifest V3' },
      { key: 'name', desc: 'Extension name' },
      { key: 'version', desc: 'Version' },
      { key: 'description', desc: 'Description' },
      { key: 'permissions', desc: 'Permissions array' },
      { key: 'host_permissions', desc: 'Host permissions' },
      { key: 'background', desc: 'Background service worker' },
      { key: 'content_scripts', desc: 'Content scripts' },
      { key: 'action', desc: 'Action (popup)' },
      { key: 'icons', desc: 'Icons' },
    ];

    checks.forEach(check => {
      if (check.value !== undefined) {
        if (manifest[check.key] === check.value) {
          success.push(`✓ Manifest: ${check.desc} = ${check.value}`);
        } else {
          errors.push(`✗ Manifest: ${check.desc} incorrect (expected ${check.value})`);
        }
      } else {
        if (manifest[check.key]) {
          success.push(`✓ Manifest: ${check.desc} configured`);
        } else {
          errors.push(`✗ Manifest: ${check.desc} missing`);
        }
      }
    });

  } catch (error) {
    errors.push(`✗ Cannot parse manifest.json: ${error.message}`);
  }
}

function checkBuild() {
  try {
    const distExists = fs.existsSync(path.join(projectRoot, 'dist'));
    if (distExists) {
      success.push(`✓ Build artifacts exist (dist/ directory)`);

      const requiredFiles = [
        'dist/manifest.json',
        'dist/background.js',
        'dist/content.js',
        'dist/popup/popup.html',
        'dist/sidebar/sidebar.html',
        'dist/settings/settings.html',
      ];

      requiredFiles.forEach(file => {
        if (fs.existsSync(path.join(projectRoot, file))) {
          success.push(`  ✓ ${file}`);
        } else {
          warnings.push(`  ⚠ Missing: ${file}`);
        }
      });
    } else {
      errors.push(`✗ Build artifacts missing. Run: npm run build`);
    }
  } catch (error) {
    errors.push(`✗ Build check failed: ${error.message}`);
  }
}

function checkIcons() {
  const iconSizes = [16, 48, 128, 512];
  const iconDir = path.join(projectRoot, 'images');

  if (!fs.existsSync(iconDir)) {
    errors.push(`✗ Icons directory missing (images/)`);
    return;
  }

  let hasAll = true;
  iconSizes.forEach(size => {
    const iconPath = path.join(iconDir, `icon-${size}.png`);
    if (fs.existsSync(iconPath)) {
      success.push(`✓ Icon ${size}x${size} exists`);
    } else {
      warnings.push(`⚠ Icon ${size}x${size} missing (need to generate from icon.svg)`);
      hasAll = false;
    }
  });

  if (fs.existsSync(path.join(iconDir, 'icon.svg'))) {
    success.push(`✓ Icon SVG source available`);
    if (!hasAll) {
      warnings.push(`  → Convert to PNG: https://convertio.co/svg-png/`);
    }
  }
}

function checkDocumentation() {
  const docs = [
    { file: 'README.md', desc: 'README documentation' },
    { file: 'CHROME_WEBSTORE_SUBMISSION.md', desc: 'Chrome Web Store submission guide' },
    { file: 'LAUNCH_GUIDE.md', desc: 'User launch guide' },
    { file: 'BUNDLE_OPTIMIZATION.md', desc: 'Bundle optimization report' },
  ];

  docs.forEach(doc => {
    checkFile(doc.file, doc.desc);
  });
}

function checkPackageJson() {
  try {
    const pkg = JSON.parse(fs.readFileSync(path.join(projectRoot, 'package.json'), 'utf8'));

    const deps = Object.keys(pkg.dependencies || {});
    if (deps.includes('react') && deps.includes('zustand')) {
      success.push(`✓ Core dependencies present (React, Zustand)`);
    }

    const removed = ['tailwindcss', 'tweetnacl', 'autoprefixer', 'postcss'];
    const hasRemoved = removed.filter(d => deps.includes(d));
    if (hasRemoved.length === 0) {
      success.push(`✓ Unused dependencies removed`);
    }

  } catch (error) {
    warnings.push(`⚠ Cannot parse package.json: ${error.message}`);
  }
}

function checkTests() {
  const testFiles = [
    'src/__tests__/e2e.test.ts',
    'src/__tests__/integration.test.ts',
    'src/api/__tests__/providers.test.ts',
    'src/stores/__tests__/conversationStore.test.ts',
  ];

  let count = 0;
  testFiles.forEach(file => {
    if (fs.existsSync(path.join(projectRoot, file))) {
      count++;
    }
  });

  if (count >= 4) {
    success.push(`✓ Comprehensive test suite (${count}+ test files)`);
  } else {
    warnings.push(`⚠ Limited test coverage (${count} test files found)`);
  }
}

function printReport() {
  console.log('\n' + '='.repeat(60));
  console.log('MOLY - CHROME WEB STORE SUBMISSION VERIFICATION');
  console.log('='.repeat(60) + '\n');

  // Print success items
  if (success.length > 0) {
    console.log('\n✅ PASSED CHECKS:');
    console.log('─'.repeat(60));
    success.forEach(item => console.log(item));
  }

  // Print warnings
  if (warnings.length > 0) {
    console.log('\n⚠️  WARNINGS:');
    console.log('─'.repeat(60));
    warnings.forEach(item => console.log(item));
  }

  // Print errors
  if (errors.length > 0) {
    console.log('\n❌ ERRORS:');
    console.log('─'.repeat(60));
    errors.forEach(item => console.log(item));
  }

  // Summary
  console.log('\n' + '='.repeat(60));
  console.log(`SUMMARY: ${success.length} passed, ${warnings.length} warnings, ${errors.length} errors`);
  console.log('='.repeat(60) + '\n');

  if (errors.length > 0) {
    console.log('❌ SUBMISSION NOT READY');
    console.log('Fix the above errors before submitting.\n');
    process.exit(1);
  } else if (warnings.length > 0) {
    console.log('⚠️  SUBMISSION READY WITH WARNINGS');
    console.log('Please review warnings before submitting.\n');
    process.exit(0);
  } else {
    console.log('✅ READY FOR SUBMISSION!');
    console.log('All checks passed. Extension is ready for Chrome Web Store.\n');
    process.exit(0);
  }
}

// Run all checks
console.log('\nRunning pre-submission verification...\n');

console.log('Checking manifest configuration...');
checkManifest();

console.log('Checking build artifacts...');
checkBuild();

console.log('Checking icon assets...');
checkIcons();

console.log('Checking documentation...');
checkDocumentation();

console.log('Checking dependencies...');
checkPackageJson();

console.log('Checking test coverage...');
checkTests();

console.log('Checking required files...');
checkFile('dist/manifest.json', 'Built manifest');
checkFile('package.json', 'Package configuration');
checkFile('tsconfig.json', 'TypeScript configuration');

// Print detailed report
printReport();
