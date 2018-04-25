interface FileSaver {
    (
        /**
         * @summary Data.
         * @type {Blob}
         */
        data: Blob,

        /**
         * @summary File name.
         * @type {DOMString}
         */
        filename: string,

        /**
         * @summary Disable Unicode text encoding hints or not.
         * @type {boolean}
         */
        disableAutoBOM?: boolean
    ): void
}

declare var saveAs: FileSaver;

declare module "file-saver" {
    export = saveAs
}
