console.log('=== Starting test program ===');
console.log('Hello from Node.js!');
console.log('Current working directory:', process.cwd());
console.log('Node.js version:', process.version);
console.log('Platform:', process.platform);

setTimeout(() => {
  console.log('After 1 second delay...');
}, 1000);

setTimeout(() => {
  console.log('After 2 seconds delay...');
}, 2000);

setTimeout(() => {
  console.log('=== Program ending ===');
  process.exit(0);
}, 3000);

console.log('=== Scheduled all timeouts ===');console.log('=== Starting test program ===');
console.log('Hello from Node.js!');
console.log('Current working directory:', process.cwd());
console.log('Node.js version:', process.version);
console.log('Platform:', process.platform);

setTimeout(() => {
  console.log('After 1 second delay...');
}, 1000);

setTimeout(() => {
  console.log('After 2 seconds delay...');
}, 2000);

setTimeout(() => {
  console.log('=== Program ending ===');
  process.exit(0);
}, 3000);

console.log('=== Scheduled all timeouts ===');