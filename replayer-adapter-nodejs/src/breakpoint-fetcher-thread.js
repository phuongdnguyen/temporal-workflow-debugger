/**
 * Worker thread entrypoint for fetching breakpoints from IDE.
 * This allows making HTTP requests outside the workflow sandbox.
 */
const { isMainThread, parentPort } = require('node:worker_threads');
const { httpGet } = require('./http-client');

/**
 * Request from parent thread to fetch breakpoints
 */
// interface BreakpointRequest {
//   type: 'fetch-breakpoints';
//   debuggerAddr: string;
//   responseBuffer: Int32Array;
//   dataBuffer: SharedArrayBuffer;
// }

if (isMainThread) {
  throw new Error(`Imported ${__filename} from main thread`);
}

if (!parentPort) {
  throw new TypeError(`${__filename} got a null parentPort`);
}

parentPort.on('message', async (request) => {
  const { debuggerAddr, responseBuffer, dataBuffer } = request;
  
  try {
    // Fetch breakpoints from IDE using existing http-client
    const response = await httpGet(`${debuggerAddr}/breakpoints`);
    
    if (response.statusCode !== 200) {
      throw new Error(`HTTP error! status: ${response.statusCode}, body: ${response.body}`);
    }

    const data = JSON.parse(response.body);
    console.log(`Worker thread fetched breakpoints: ${response.body}`);
    
    // Handle different response formats (same logic as current implementation)
    let breakpointIds = [];
    if (Array.isArray(data)) {
      breakpointIds = data;
    } else if (data.breakpoints && Array.isArray(data.breakpoints)) {
      breakpointIds = data.breakpoints;
    } else if (data.eventIds && Array.isArray(data.eventIds)) {
      breakpointIds = data.eventIds;
    } else {
      console.warn('Unexpected breakpoints response format:', data);
      breakpointIds = [];
    }

    // Filter to ensure we only have numbers
    breakpointIds = breakpointIds.filter(id => typeof id === 'number');
    
    // Store breakpoint data in shared buffer
    const breakpointData = JSON.stringify(breakpointIds);
    const encoded = Buffer.from(breakpointData, 'utf8');
    
    // Write data to shared buffer (leave first 4 bytes for length)
    const dataView = new Uint8Array(dataBuffer, 4);
    const maxLength = dataBuffer.byteLength - 4;
    const copyLength = Math.min(encoded.length, maxLength);
    
    for (let i = 0; i < copyLength; i++) {
      dataView[i] = encoded[i];
    }
    
    // Store length in first 4 bytes
    const lengthView = new Uint32Array(dataBuffer, 0, 1);
    lengthView[0] = copyLength;
    
    // Signal success
    Atomics.store(responseBuffer, 0, 1);
  } catch (err) {
    console.error('Worker thread failed to fetch breakpoints:', err);
    Atomics.store(responseBuffer, 0, 2); // Error
  } finally {
    Atomics.notify(responseBuffer, 0, 1);
  }
}); 