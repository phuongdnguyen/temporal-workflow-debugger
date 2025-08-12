/**
 * Worker thread entrypoint for fetching breakpoints from IDE.
 * This allows making HTTP requests outside the workflow sandbox.
 */
const { isMainThread, parentPort } = require('node:worker_threads');
const { httpGet, httpPost } = require('./http-client');

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
  const { type } = request;

  // Ensure we always notify the waiting thread
  const safeNotify = (responseBuffer, status) => {
    try {
      if (responseBuffer) {
        Atomics.store(responseBuffer, 0, status);
        Atomics.notify(responseBuffer, 0, 1);
      }
    } catch (notifyErr) {
      console.error('Worker thread failed to notify parent:', notifyErr);
    }
  };

  try {
    if (type === 'fetch-breakpoints') {
      const { debuggerAddr, responseBuffer, dataBuffer } = request;

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
      breakpointIds = breakpointIds.filter((id) => typeof id === 'number');

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
      safeNotify(responseBuffer, 1);
      return;
    }

    if (type === 'highlight-event') {
      const { debuggerAddr, eventId, responseBuffer } = request;
      console.log(`worker thread received highlight-event request, debuggerAddr: ${debuggerAddr}, eventId: ${eventId}`)

      const payload = JSON.stringify({ "eventId": eventId });
      // Keep endpoint aligned with manager's previous implementation
      const url = `${debuggerAddr}/current-event`;
      const response = await httpPost(url, payload);

      console.log(
        `Worker thread highlight response status: ${response.statusCode}, body: ${response.body}`
      );

      if (response.statusCode !== 200) {
        throw new Error(`Highlight request failed: ${response.statusCode} ${response.body}`);
      }

      safeNotify(responseBuffer, 1);
      return;
    }

    console.warn('Worker thread received unknown message type:', type);
    safeNotify(request.responseBuffer, 2);
  } catch (err) {
    console.error('Worker thread error handling request:', err);
    safeNotify(request.responseBuffer, 2);
  }
});