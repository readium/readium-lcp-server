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

const URL = 'http://localhost:8991/PublicationUpload';
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

    hideFilename: boolean = false;
    submitButtonLabel: string = "Add";
    form: FormGroup;

    snackMessage: string = "";
    newPublication:boolean = true;

    fileName: string;

    public uploader:FileUploader = new FileUploader({url: URL});
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
        this.lastFile = fileItem;
    }

    ngOnInit(): void {
        this.refreshMasterFiles();

        // Events declarations
        this.uploader.onAfterAddingFile = (fileItem: any) => {this.onItemAdded(fileItem)}
        this.uploader.onCompleteAll = () => {this.AllUploaded()}

        if (this.publication == null) {
            this.submitButtonLabel = "Add";
            this.form = this.fb.group({
                "title": ["", Validators.required],
                "filename": ["", Validators.required]
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
            this.lastFile.file.name = this.fileName;
            this.newPublication = true;
            if (confirm){
                this.publicationService.checkByName(this.form.value['title']).then(
                    result => {
                        if (result == 0)
                        {
                            this.uploader.uploadItem(this.lastFile);
                        }
                        else
                        {
                            this.showSnackBar();
                        }
                    }
                );
            } else {
                this.newPublication = false;
                this.uploader.uploadItem(this.lastFile);
                this.gotoList();
            }
        }
    }

    // When all the files are uploaded, create the publication
    AllUploaded(): void
    {
        if (this.newPublication){
            // Create publication
            let publication = new Publication();
            publication.title = this.form.value['title'];
            publication.masterFilename = this.fileName;
            this.publicationService.add(publication)
            .then(
                newPublication => {
                    this.gotoList();
                }
            );
        }
    }

    showSnackBar() {
        var x = $("#snackbar");
        x.attr("class","show stay");
    }

    hideSnackBar() {
        var x = $("#snackbar");
        x.attr("class","");
    }
}