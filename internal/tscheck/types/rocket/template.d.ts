export type TaggedLiteralScope = {
    signalPathBase: string;
    localSignals?: Record<string, string>;
};
export type TaggedLiteral = (strings: TemplateStringsArray, ...values: unknown[]) => DocumentFragment;
export declare const appendComposedTemplateValue: (fragment: DocumentFragment, value: unknown) => void;
export declare function hypertext(render: (string: string) => ParentNode, postprocess: (root: ParentNode) => DocumentFragment): ({ raw: strings }: TemplateStringsArray, ...values: unknown[]) => DocumentFragment;
export declare const signalNameAttributes: Set<string>;
export declare const rocketDispatchActionName = "dispatchRocket";
export declare const rocketRefAttr = "data-rocket-ref";
export declare const rewriteDataAttributes: (root: ParentNode, scope?: TaggedLiteralScope) => void;
