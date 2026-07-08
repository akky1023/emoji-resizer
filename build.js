const { execSync } = require('child_process');
const pkg = require('./package.json');

// Prefix version with 'v' (e.g. v1.0.0)
const version = `v${pkg.version}`;

console.log(`Building emoji-resizer with version ${version}...`);

try {
  // Execute the Go build injecting the version from package.json
  execSync(`go build -ldflags "-X main.version=${version}" -o emoji-resizer.exe src/main.go`, {
    stdio: 'inherit'
  });
  console.log('Build successful!');
} catch (error) {
  console.error('Build failed:', error.message);
  process.exit(1);
}
