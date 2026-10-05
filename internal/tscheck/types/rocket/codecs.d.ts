type DefaultValue<T> = T | (() => T);
export type Codec<T> = {
    decode(value: unknown): T;
    encode(value: T): string;
};
export type CodecDocs = {
    description?: string;
    label?: string;
    control?: 'auto' | 'text' | 'textarea' | 'number' | 'boolean' | 'select';
    placeholder?: string;
};
export type CodecManifest = {
    type: 'string' | 'number' | 'boolean' | 'date' | 'json' | 'js' | 'binary' | 'array' | 'tuple' | 'object' | 'oneOf' | 'custom';
    values?: readonly unknown[];
    docs?: CodecDocs;
};
type RuntimeCodec<T> = Codec<T> & {
    decode(value: unknown): T;
    encode(value: T): string;
    default(value: DefaultValue<T>): RuntimeCodec<T>;
    docs(meta: CodecDocs): RuntimeCodec<T>;
    readonly manifestMeta?: CodecManifest;
};
export type InferCodec<T> = T extends RuntimeCodec<infer Value> ? Value : never;
export type StringCodec = RuntimeCodec<string> & {
    readonly trim: StringCodec;
    readonly upper: StringCodec;
    readonly lower: StringCodec;
    readonly kebab: StringCodec;
    readonly camel: StringCodec;
    readonly snake: StringCodec;
    readonly pascal: StringCodec;
    readonly title: StringCodec;
    prefix(value: string): StringCodec;
    suffix(value: string): StringCodec;
    maxLength(length: number): StringCodec;
    default(value: DefaultValue<string>): StringCodec;
};
export type NumberCodec = RuntimeCodec<number> & {
    min(value: number): NumberCodec;
    max(value: number): NumberCodec;
    clamp(minValue: number, maxValue: number): NumberCodec;
    step(stepValue: number, base?: number): NumberCodec;
    readonly round: NumberCodec;
    ceil(decimals?: number): NumberCodec;
    floor(decimals?: number): NumberCodec;
    fit(inMin: number, inMax: number, outMin: number, outMax: number, clamped?: boolean, rounded?: boolean): NumberCodec;
    default(value: DefaultValue<number>): NumberCodec;
};
export type BoolCodec = RuntimeCodec<boolean> & {
    default(value: DefaultValue<boolean>): BoolCodec;
};
export type DateCodec = RuntimeCodec<Date> & {
    default(value: DefaultValue<Date>): DateCodec;
};
export type JsonCodec<T = any> = RuntimeCodec<T> & {
    default(value: DefaultValue<T>): JsonCodec<T>;
};
export type JsCodec<T = any> = RuntimeCodec<T> & {
    default(value: DefaultValue<T>): JsCodec<T>;
};
export type BinCodec = RuntimeCodec<Uint8Array> & {
    default(value: DefaultValue<Uint8Array>): BinCodec;
};
export type ArrayCodec<T> = RuntimeCodec<T[]> & {
    default(value: DefaultValue<T[]>): ArrayCodec<T>;
};
export type TupleCodec<T extends readonly unknown[]> = RuntimeCodec<T> & {
    default(value: DefaultValue<T>): TupleCodec<T>;
};
export type ObjectCodec<T extends Record<string, any>> = RuntimeCodec<T> & {
    default(value: DefaultValue<T>): ObjectCodec<T>;
};
export type OneOfCodec<T> = RuntimeCodec<T> & {
    default(value: DefaultValue<T>): OneOfCodec<T>;
};
export type CodecRegistry = {
    string: StringCodec;
    number: NumberCodec;
    bool: BoolCodec;
    date: DateCodec;
    json: JsonCodec;
    js: JsCodec;
    bin: BinCodec;
    array<T>(codec: RuntimeCodec<T>): ArrayCodec<T>;
    array<T extends readonly RuntimeCodec<any>[]>(...codecs: T): TupleCodec<{
        [K in keyof T]: InferCodec<T[K]>;
    }>;
    object<T extends Record<string, RuntimeCodec<any>>>(shape: T): ObjectCodec<{
        [K in keyof T]: InferCodec<T[K]>;
    }>;
    oneOf<const T extends readonly unknown[]>(...values: T): OneOfCodec<T[number]>;
    oneOf<T extends readonly RuntimeCodec<any>[]>(...codecs: T): OneOfCodec<InferCodec<T[number]>>;
};
export type PropDefs = Record<string, RuntimeCodec<any>>;
export type InferProps<T extends PropDefs> = {
    [K in keyof T]: InferCodec<T[K]>;
};
export declare const getCodecDefault: <T>(codec: RuntimeCodec<T>) => any;
export declare const decodeCodec: <T>(codec: RuntimeCodec<T>, value: unknown) => T;
export declare const createCodec: <T>(handlers: Codec<T>) => RuntimeCodec<T>;
export declare const getCodecManifest: <T>(codec: RuntimeCodec<T>) => CodecManifest;
export declare const codecRegistry: CodecRegistry;
export {};
