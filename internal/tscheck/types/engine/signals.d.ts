import type { Computed, Effect, JSONPatch, MergePatchArgs, Paths, Signal, SignalFilterOptions } from '../engine/types';
interface ReactiveNode {
    deps_?: Link;
    depsTail_?: Link;
    subs_?: Link;
    subsTail_?: Link;
    flags_: ReactiveFlags;
}
interface Link {
    version_: number;
    dep_: ReactiveNode;
    sub_: ReactiveNode;
    prevSub_?: Link;
    nextSub_?: Link;
    prevDep_?: Link;
    nextDep_?: Link;
}
declare enum ReactiveFlags {
    None = 0,
    Mutable = 1,
    Watching = 2,
    RecursedCheck = 4,
    Recursed = 8,
    Dirty = 16,
    Pending = 32
}
export declare const beginBatch: () => void;
export declare const endBatch: () => void;
export declare const startPeeking: (sub?: ReactiveNode) => void;
export declare const stopPeeking: () => void;
export declare const signal: <T>(initialValue?: T) => Signal<T>;
export declare const computed: <T>(getter: (previousValue?: T) => T) => Computed<T>;
export declare const effect: (fn: () => void) => Effect;
export declare const getPath: <T = any>(path: string) => T | undefined;
export declare const mergePatch: (patch: JSONPatch, { ifMissing }?: MergePatchArgs) => void;
export declare const mergePaths: (paths: Paths, options?: MergePatchArgs) => void;
/**
 * Filters the root store based on an include and exclude RegExp
 *
 * @returns The filtered object
 */
export declare const filtered: ({ include, exclude }?: SignalFilterOptions, obj?: JSONPatch) => Record<string, any>;
export declare const root: Record<string, any>;
export {};
