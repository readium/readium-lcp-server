import { Component, Input, OnInit, Directive }                  from '@angular/core';
import { Router }                                               from '@angular/router';
import {
    FormGroup,
    FormControl,
    Validators,
    FormBuilder }                                               from '@angular/forms';

import { Publication }                                          from './publication';
import { PublicationService }                                   from './publication.service';
import { MasterFile }                                           from './master-file';
import { FileSelectDirective, FileDropDirective, FileUploader } from 'ng2-file-upload';

declare var Config: any;

@Component({
    moduleId: module.id,
    selector: 'lcp-publication-form',
    templateUrl: 'publication-form.component.html'
})

export class PublicationFormComponent implements OnInit {
    @Input()
    publication: Publication;
    masterFiles: MasterFile[];
    baseUrl: string = Config.frontend.url;

    hideFilename: boolean = false;
    submitButtonLabel: string = "Add";
    form: FormGroup;

    snackMessage: string = "";
    newPublication: boolean = true;
    uploadConfimation: boolean;
    errorMessage: string = "";

    fileName: string;

    public uploader:FileUploader;
    public lastFile:any;
    public hasBaseDropZoneOver:boolean = false;
    public notAnEPUB: boolean = false;

    constructor(
        private fb: FormBuilder,
        private router: Router,
        private publicationService: PublicationService) {
    }

    public fileOverBase(e:any):void {
        this.hasBaseDropZoneOver = e;
    }

    refreshMasterFiles(): void {
        this.publicationService.getMasterFiles().then(
            masterFiles => {
                this.masterFiles = masterFiles;
            }
        );
    }

    onItemAdded = function(fileItem: any)
    {
        this.split = fileItem.file.name.split('.');
        if (this.split[this.split.length-1] === "epub")
        {
            this.notAnEPUB = false;
        }
        else
        {
            this.notAnEPUB = true;
        }
        this.uploader.queue = [fileItem];
        let publication : Publication = new Publication();
        publication.title = this.form.value['title']
        publication.masterFilename = this.fileName;
        this.lastFile = fileItem;
    }

    ngOnInit(): void {
        this.uploader = new FileUploader({url: this.baseUrl + "/PublicationUpload"});
        this.refreshMasterFiles();

        // Events declarations
        this.uploader.onAfterAddingFile = (fileItem: any) => {this.onItemAdded(fileItem)}
        this.uploader.onCompleteAll = () => {this.AllUploaded()}

        if (this.publication == null) {
            this.submitButtonLabel = "Add";
            this.form = this.fb.group({
                "title": ["", Validators.required],
                "filename": ["", Validators.required],
                "type": ["UPLOAD", Validators.nullValidator]
            });
        } else {
            this.hideFilename = true
            this.submitButtonLabel = "Save";
            this.form = this.fb.group({
                "title": [this.publication.title, Validators.required]
            });
        }
    }

    gotoList() {
        this.router.navigate(['/publications']);
    }

    onCancel() {
        this.gotoList();
    }

    onSubmit(confirm: boolean) {
        if (this.publication) {
            // Update publication
            this.publication.title = this.form.value['title'];

            this.publicationService.update(
                this.publication
            ).then(
                publication => {
                    this.gotoList();
                }
            );

        } else {
            this.fileName = this.form.value['title'] + '.epub';
            if (this.form.value["type"] === "UPLOAD") {
                this.lastFile.file.name = this.fileName;
            }
            this.newPublication = true;
            if (confirm) {
                this.publicationService.checkByName(this.form.value['title']).then(
                    result => {
                        if (result === 0) {
                            if (this.form.value["type"] === "UPLOAD") {
                                let options = {url: this.baseUrl + "/PublicationUpload?title=" + this.form.value['title']};
                                this.uploader.setOptions(options);
                                this.uploader.uploadItem(this.lastFile);
                            } else {
                                let publication = new Publication();
                                publication.title = this.form.value['title'];
                                publication.masterFilename = this.form.value['filename'];

                                this.publicationService.addPublication(publication)
                                .then( error => {
                                    console.log(error);
                                        this.uploadConfimation = false;
                                        if (error === 200) {
                                            this.gotoList();
                                        } else if (error === 400) {
                                            this.errorMessage = "The file must be a real epub file."
                                            this.showSnackBar(false);
                                        }
                                    }
                                );
                            }
                        } else {
                            this.uploadConfimation = true;
                            this.showSnackBar(true);
                        }
                    }
                );
            } else {
                this.newPublication = false;
                if (this.form.value["type"] === "UPLOAD") {
                    let options = {url: this.baseUrl + "/PublicationUpload?title=" + this.form.value['title']};
                    this.uploader.setOptions(options);
                    this.uploader.uploadItem(this.lastFile);
                } else {
                    this.AllUploaded();
                }
                this.gotoList();
            }
        }
    }

    // When all the files are uploaded, create the publication
    AllUploaded(): void {
        this.gotoList();
    }

    showSnackBar(stay: boolean) {
        var snakeClass: string = "show stay";

        if (!stay) snakeClass = "show";
        var x = $("#snackbar");
        x.attr("class",snakeClass);
        if (!stay) setTimeout(function(){$("#snackbar").attr("class","");}, 3000);
    }

    hideSnackBar() {
        var x = $("#snackbar");
        x.attr("class","");
    }
}
