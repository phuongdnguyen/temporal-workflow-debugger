/**
 * Activity interceptors for replayer adapter debugging.
 * These interceptors notify the runner when activity operations occur,
 * enabling breakpoint support for activity execution during replay.
 */

import { ActivityInterceptorsFactory } from '@temporalio/worker';
import { raiseSentinelBreakpoint } from './replayer';

/**
 * Factory function that creates interceptors for activity replay debugging
 */
export const activityInterceptors: ActivityInterceptorsFactory = () => ({
  inbound: {
    async execute(input, next) {
      raiseSentinelBreakpoint('ExecuteActivity', null);
      return next(input);
    },
  },
  outbound: {
    getLogAttributes(input, next) {
      raiseSentinelBreakpoint('GetLogAttributes', null);
      return next(input);
    },
    getMetricTags(input, next) {
      raiseSentinelBreakpoint('GetMetricTags', null);
      return next(input);
    },
  },
}); 