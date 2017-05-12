import { NgModule }             from '@angular/core';
import { CommonModule }         from '@angular/common';
import { RouterModule }         from '@angular/router';
import {
    FormsModule,
    ReactiveFormsModule }       from '@angular/forms';

import { UserService }          from './user.service';
import { UserRoutingModule }    from './user-routing.module';
import { UserListComponent }    from './user-list.component';
import { UserFormComponent }    from './user-form.component';
import { UserAddComponent }     from './user-add.component';
import { UserEditComponent }    from './user-edit.component';
import { SortModule }           from '../shared/pipes/sort.module';

@NgModule({
    imports: [
        CommonModule,
        RouterModule,
        FormsModule,
        ReactiveFormsModule,
        UserRoutingModule,
        SortModule
    ],
    declarations: [
        UserListComponent,
        UserFormComponent,
        UserAddComponent,
        UserEditComponent
    ],
    providers: [
        UserService
    ]
})

export class UserModule { }
