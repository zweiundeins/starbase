import type { SignalFilterOptions } from '../../engine/types';
export declare const STARTED = "started";
export declare const FINISHED = "finished";
export declare const ERROR = "error";
export declare const RETRYING = "retrying";
export declare const RETRIES_FAILED = "retries-failed";
type ResponseOverrides = {
    selector?: string;
    mode?: string;
    namespace?: string;
    useViewTransition?: boolean;
} | {
    onlyIfMissing?: boolean;
};
export type FetchArgs = {
    selector?: string;
    headers?: Record<string, string>;
    contentType?: 'json' | 'form';
    filterSignals?: SignalFilterOptions;
    openWhenHidden?: boolean;
    payload?: any;
    requestCancellation?: 'auto' | 'cleanup' | 'disabled' | AbortController;
    responseOverrides?: ResponseOverrides;
    retry?: 'auto' | 'error' | 'always' | 'never';
    retryInterval?: number;
    retryScaler?: number;
    retryMaxWait?: number;
    retryMaxCount?: number;
};
export {};
