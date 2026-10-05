import type { ActionContext, ActionPlugin, AttributePlugin, HTMLOrSVG, Modifiers, Requirement, WatcherPlugin } from '../engine/types';
export declare const actions: Record<string, (ctx: ActionContext, ...args: any[]) => any>;
export declare const attribute: <R extends Requirement, B extends boolean>(plugin: AttributePlugin<R, B>) => void;
export declare const action: <T>(plugin: ActionPlugin<T>) => void;
export declare const watcher: (plugin: WatcherPlugin) => void;
export declare const cleanupTree: (root: ParentNode) => void;
export declare const parseAttributeKey: (rawKey: string) => {
    pluginName: string;
    key: string | undefined;
    mods: Modifiers;
};
export declare const isDocumentObserverActive: () => boolean;
export declare const apply: (root?: HTMLOrSVG | ShadowRoot, observeRoot?: boolean) => void;
export declare const forgetRoot: (root: HTMLOrSVG | ShadowRoot) => void;
export declare const applyElement: (el: HTMLOrSVG, onlyNew?: boolean) => void;
type GenRxOptions = {
    returnsValue_?: boolean;
    argNames_?: string[];
    cleanups_?: Map<string, () => void>;
};
type GenRxFn = <T>(el: HTMLOrSVG, ...args: any[]) => T;
export declare const genRx: (value: string, { returnsValue_: returnsValue, argNames_: argNames, cleanups_: cleanups, }?: GenRxOptions) => GenRxFn;
export {};
