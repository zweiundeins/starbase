import type { HTMLOrSVG } from '../engine/types';
import { type CodecRegistry, type InferProps, type PropDefs } from './codecs';
import { type TaggedLiteral } from './template';
export declare const rocketHostAttr = "data-rocket-host";
type AnyRecord = {
    [key: string]: any;
};
type SetupSignal = (<T>(name: string, initialValue: T) => T) & AnyRecord;
export type RefCtors = Record<string, abstract new (...args: any[]) => Element>;
export type InstancesOf<C extends RefCtors> = {
    [K in keyof C]?: InstanceType<C[K]>;
};
type StateRecord = AnyRecord;
type PropOverrideGetter<Props extends Record<string, any>, Name extends keyof Props & string> = (getDefault: () => Props[Name]) => any;
type PropOverrideSetter<Props extends Record<string, any>, Name extends keyof Props & string> = (value: any, setDefault: (value: Props[Name]) => void) => void;
type HostPropDescriptor = Omit<PropertyDescriptor, 'configurable'>;
export type RocketHost = HTMLElement & {
    dispatchRocketAction(name: string, el: Element | null, evt: Event | undefined, cleanups: Map<string, () => void>, ...args: any[]): any;
};
export type RocketHostWithProps<Props extends Record<string, any>> = RocketHost & Props;
type SetupEmit = {
    (type: string): void;
    (...types: [string, ...string[]]): void;
    <Detail>(type: string, detail: Detail, options?: Omit<CustomEventInit<Detail>, 'detail'>): void;
};
type SetupEmitCancellable = {
    (type: string): boolean;
    <Detail>(type: string, detail: Detail, options?: Omit<CustomEventInit<Detail>, 'detail' | 'cancelable'>): boolean;
};
type PropObserver<Props extends Record<string, any>> = (() => void) | ((props: Props, changes: Partial<Props>) => void);
type SetupContext<Props extends Record<string, any>> = {
    props: Props;
    $: Record<string, any>;
    $$: SetupSignal;
    effect(fn: () => void): () => void;
    apply(root: HTMLOrSVG | ShadowRoot, merge?: boolean): void;
    adoptStyles(host: HTMLElement, ...styles: string[]): void;
    cleanup(fn: () => void): void;
    emit: SetupEmit;
    emitCancellable: SetupEmitCancellable;
    actions: Record<string, (...args: any[]) => any>;
    action(name: string, fn: RocketAction<Props>): void;
    observeProps(fn: PropObserver<Props>, ...propNames: Array<keyof Props & string>): () => void;
    overrideProp<Name extends keyof Props & string>(name: Name, getter?: PropOverrideGetter<Props, Name>, setter?: PropOverrideSetter<Props, Name>): void;
    defineHostProp(name: string, descriptor: HostPropDescriptor): void;
    internals: ElementInternals;
    onFormReset(fn: () => void): void;
    onFormDisabled(fn: (disabled: boolean) => void): void;
    onFormAssociated(fn: (form: HTMLFormElement | null) => void): void;
    onFormStateRestore(fn: (state: unknown, mode: 'restore' | 'autocomplete') => void): void;
    render: SetupRender<Props>;
    host: RocketHostWithProps<Props>;
};
type FirstUpdateContext<Props extends Record<string, any>, Refs extends RefCtors = RefCtors> = SetupContext<Props> & {
    refs: InstancesOf<Refs>;
};
type RenderContext<Props extends Record<string, any>> = {
    html: TaggedLiteral;
    svg: TaggedLiteral;
    props: Props;
    host: RocketHostWithProps<Props>;
};
type RenderContextOverrides<Props extends Record<string, any>> = Partial<RenderContext<Props>>;
type RocketPrimitiveRenderValue = string | number | boolean | bigint | Date | null | undefined;
type RocketComposedRenderValue = RocketPrimitiveRenderValue | Node | Iterable<RocketComposedRenderValue>;
type RocketRenderValue = DocumentFragment | RocketPrimitiveRenderValue | Iterable<RocketComposedRenderValue>;
type RocketRender<Props extends Record<string, any>> = (context: RenderContext<Props>, ...args: any[]) => RocketRenderValue;
type SetupRender<Props extends Record<string, any>> = (context: RenderContextOverrides<Props>, ...args: any[]) => void;
type RocketAction<Props extends Record<string, any>> = (context: {
    host: RocketHostWithProps<Props>;
    props: Props;
    state: StateRecord;
    el: Element | null;
    evt: Event | undefined;
}, ...args: any[]) => any;
type RocketDefinition<Defs extends PropDefs = PropDefs, Refs extends RefCtors = RefCtors> = {
    refs?: Refs;
    props?: (codecs: CodecRegistry) => Defs;
    manifest?: RocketManifestMeta;
    setup?: (context: SetupContext<InferProps<Defs>>) => void;
    onFirstRender?: (context: FirstUpdateContext<InferProps<Defs>, Refs>) => void;
    render?: RocketRender<InferProps<Defs>>;
    mode?: 'open' | 'closed' | 'light';
    formAssociated?: boolean;
    delegatesFocus?: boolean;
    renderOnPropChange?: boolean | ((context: {
        host: RocketHostWithProps<InferProps<Defs>>;
        props: InferProps<Defs>;
        changes: Partial<InferProps<Defs>>;
    }) => boolean);
};
type RocketManifestMeta = {
    slots?: RocketManifestSlot[];
    events?: RocketManifestEvent[];
};
type RocketManifestSlot = {
    name: string;
    description?: string;
};
type RocketManifestEvent = {
    name: string;
    kind?: 'event' | 'custom-event';
    bubbles?: boolean;
    composed?: boolean;
    description?: string;
};
type RocketPublishOptions = {
    endpoint: string;
    headers?: Record<string, string>;
};
export declare const publishRocketManifests: ({ endpoint, headers, }: RocketPublishOptions) => Promise<Response>;
export declare function rocket<Refs extends RefCtors = RefCtors, Defs extends PropDefs = PropDefs>(tag: string, options?: RocketDefinition<Defs, Refs>): CustomElementConstructor | undefined;
export type { RocketDefinition, RocketRenderValue };
