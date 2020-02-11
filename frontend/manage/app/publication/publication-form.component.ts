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
    uploadConfirmation: boolean;
    errorMessage: string = "";

    fileName: string;

    public uploader:FileUploader;
    public droppedItem:any;
    public hasBaseDropZoneOver:boolean = false;
    public notAPublication: boolean = false;

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

    // onItemAdded is executed when a file is added to the opload component
    onItemAdded = function(fileItem: any)
    {
        this.split = fileItem.file.name.split('.');
        let extension = this.split[this.split.length-1];
        if (extension === "epub" || extension === "pdf")
        {
            this.notAPublication = false;
        }
        else
        {
            this.notAPublication = true;
        }
        this.uploader.queue = [fileItem];
        this.droppedItem = fileItem;
    }

    ngOnInit(): void {
        this.uploader = new FileUploader({url: this.baseUrl + "/publicationUpload"});        
        this.refreshMasterFiles();

        // Events declarations
        this.uploader.onAfterAddingFile = (fileItem: any) => {this.onItemAdded(fileItem); fileItem.withCredentials = false; }
        this.uploader.onCompleteAll = () => {this.gotoList()}

        // case = import of a new publication
        if (this.publication == null) {
            this.submitButtonLabel = "Add";
            this.form = this.fb.group({
                "title": ["", Validators.required],
                "filename": ["", Validators.required],
                "type": ["UPLOAD", Validators.nullValidator]
            });
        // case = edition of an existing publication
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

    // onSubmit imports a publication into the frontend server, 
    // or updates information on an existing publication.
    // confirm indicates if the user must provide a confirmation in case a publication
    // already exists with the same title.  
    onSubmit(confirm: boolean) {
        // case = edition of an existing publication
        if (this.publication) {
            this.publication.title = this.form.value['title'];
            this.publicationService.update(
                this.publication
            ).then(
                publication => {
                    this.gotoList();
                }
            );
        // case = import of a new publication
        } else {
            // if the import into the frontend server needs confirmation (in case of a detected duplicate) 
            if (confirm) {
                // check the title chosen for the publication
                this.publicationService.checkByName(this.form.value['title']).then(
                    result => {
                        // if there is no duplicate
                        if (result === 0) {
                            // upload the publication
                            if (this.form.value["type"] === "UPLOAD") {
                                let options = {url: this.baseUrl + "/publicationUpload?title=" + this.form.value['title']};
                                this.uploader.setOptions(options);
                                this.uploader.uploadItem(this.droppedItem);
                            // or request the import of a publication into the frontend server
                            } else {
                                let publication = new Publication();
                                publication.title = this.form.value['title'];
                                publication.masterFilename = this.form.value['filename'];
                                this.publicationService.addPublication(publication)
                                .then( error => {
                                    console.log(error);
                                        this.uploadConfirmation = false;
                                        if (error === 200) {
                                            this.gotoList();
                                        } else if (error === 400) {
                                            this.errorMessage = "The file must be a proper EPUB or PDF file."
                                            this.showSnackBar(false);
                                        }
                                    }
                                );
                            }
                        } else {
                            this.uploadConfirmation = true;
                            this.showSnackBar(true);
                        }
                    }
                );
            // if the import into the frontend server doesn't need confirmation
            } else {
                // just treat the case of an update of the file via upload
                if (this.form.value["type"] === "UPLOAD") {
                    let options = {url: this.baseUrl + "/publicationUpload?title=" + this.form.value['title']};
                    this.uploader.setOptions(options);
                    this.uploader.uploadItem(this.droppedItem);
                }
                // the case where a new master file is selected for an existing title is not treated here
                // I could be useful, still ... 
                this.gotoList();
            }
        }
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
