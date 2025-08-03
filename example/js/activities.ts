// ====================
// ACTIVITY DEFINITIONS
// ====================

export async function greetActivity(name: string): Promise<string> {
  return `Hello, ${name}!`;
}

export async function calculateActivity(a: number, b: number): Promise<number> {
  console.log(`Calculating ${a} + ${b}`);
  await new Promise(resolve => setTimeout(resolve, 1000)); // Simulate work
  return a + b;
}

export async function processDataActivity(data: string): Promise<string> {
  console.log(`Processing data: ${data}`);
  await new Promise(resolve => setTimeout(resolve, 2000)); // Simulate work
  return `Processed: ${data.toUpperCase()}`;
} 