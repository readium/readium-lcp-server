import { Component, Input, OnInit }     from '@angular/core';
import { Router }                       from '@angular/router';
import {
    FormGroup,
    FormControl,
    Validators,
    FormBuilder }                       from '@angular/forms';

import { Publication }                  from './publication';
import { PublicationService }           from './publication.service';
import { MasterFile }                   from './master-file';

@Component({
    moduleId: module.id,
    selector: 'lcp-publication-form',
    templateUrl: 'publication-form.component.html'
})

export class PublicationFormComponent implements OnInit {
    @Input()
    publication: Publication;
    masterFiles: MasterFile[];

    hideFilename: boolean = false
    submitButtonLabel: string = "Add";
    form: FormGroup;

    private submitted = false;

    constructor(
        private fb: FormBuilder,
        private router: Router,
        private publicationService: PublicationService) {
    }

    refreshMasterFiles(): void {
        this.publicationService.getMasterFiles().then(
            masterFiles => {
                this.masterFiles = masterFiles;
            }
        );
    }

    ngOnInit(): void {
        this.refreshMasterFiles();

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

    onSubmit() {
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
            // Create publication
            let publication = new Publication();
            publication.title = this.form.value['title'];
            publication.masterFilename = this.form.value['filename'];
            console.log(publication);
            this.publicationService.add(publication)
            .then(
                newPublication => {
                    this.gotoList();
                }
            );
        }

        this.submitted = true;
    }
}
