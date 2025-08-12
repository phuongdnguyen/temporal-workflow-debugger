/**
 * Simple HTTP client for IDE communication
 */

import * as http from 'http';
import { URL } from 'url';

export interface HttpResponse {
  statusCode: number;
  body: string;
}

/**
 * Make a GET request
 */
export function httpGet(url: string, timeout = 1000): Promise<HttpResponse> {
  return new Promise((resolve, reject) => {
    const parsedUrl = new URL(url);
    const options = {
      hostname: parsedUrl.hostname,
      port: parsedUrl.port,
      path: parsedUrl.pathname + parsedUrl.search,
      method: 'GET',
      timeout,
    };

    const req = http.request(options, (res) => {
      let body = '';
      res.on('data', (chunk) => {
        body += chunk;
      });
      res.on('end', () => {
        resolve({
          statusCode: res.statusCode || 0,
          body,
        });
      });
    });

    req.on('error', reject);
    req.on('timeout', () => {
      req.destroy();
      reject(new Error('Request timeout'));
    });

    req.end();
  });
}

/**
 * Make a POST request
 */
export function httpPost(url: string, data: string, timeout = 1000): Promise<HttpResponse> {
  console.log(`http-client.httpPost url: ${url}, data: ${data}`)
  return fetch(url, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: data,
    // signal: AbortSignal.timeout(timeout),
  }).then((res) => res.text().then((body) => ({ statusCode: res.status, body })));
  return new Promise((resolve, reject) => {
    const parsedUrl = new URL(url);
    const options = {
      hostname: parsedUrl.hostname,
      port: parsedUrl.port,
      path: parsedUrl.pathname + parsedUrl.search,
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Content-Length': Buffer.byteLength(data),
      },
      timeout,
    };
    console.log(`http-client.httpPost, option: ${JSON.stringify(options)}`)

    const req = http.request(options, (res) => {
      let body = '';
      res.on('data', (chunk) => {
        body += chunk;
      });
      res.on('end', () => {
        resolve({
          statusCode: res.statusCode || 0,
          body,
        });
      });
    });

    req.on('error', reject);
    // req.on('timeout', () => {
    //   req.destroy();
    //   reject(new Error('Request timeout'));
    // });

    req.write(data);
    req.end();
  });
}
