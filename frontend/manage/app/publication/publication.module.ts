import { NgModule, CUSTOM_ELEMENTS_SCHEMA, Directive }  from '@angular/core';
import { CommonModule }                                 from '@angular/common';
import { RouterModule }                                 from '@angular/router';
import {
    FormsModule,
    ReactiveFormsModule }                               from '@angular/forms';

import { PublicationService }                           from './publication.service';
import { PublicationRoutingModule }                     from './publication-routing.module';
import { PublicationAddComponent }                      from './publication-add.component';
import { PublicationEditComponent }                     from './publication-edit.component';
import { PublicationListComponent }                     from './publication-list.component';
import { PublicationFormComponent }                     from './publication-form.component';
import { FileUploadModule }                             from 'ng2-file-upload';

@NgModule({
    imports: [
        CommonModule,
        RouterModule,
        FormsModule,
        ReactiveFormsModule,
        PublicationRoutingModule,
        FileUploadModule

    ],
    declarations: [
        PublicationAddComponent,
        PublicationEditComponent,
        PublicationListComponent,
        PublicationFormComponent
    ],
    providers: [
        PublicationService
    ],
    schemas: [ CUSTOM_ELEMENTS_SCHEMA ]
})

export class PublicationModule { }
