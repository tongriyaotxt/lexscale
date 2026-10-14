'use client';

import { useEffect, useRef, useCallback } from 'react';
import { connectSSE, SSEEvent } from '../lib/api';

/**
 * SSE Hook — 订阅后端实时事件
 *
 * 用法:
 *   useSSE((event) => {
 *     if (event.type === 'graph_updated') refetch();
 *   });
 */
export function useSSE(
  onEvent: (event: SSEEvent) => void,
  enabled = true
) {
  const callbackRef = useRef(onEvent);
  callbackRef.current = onEvent;

  const stableCallback = useCallback((event: SSEEvent) => {
    callbackRef.current(event);
  }, []);

  useEffect(() => {
    if (!enabled) return;
    const disconnect = connectSSE(stableCallback);
    return disconnect;
  }, [enabled, stableCallback]);
}
