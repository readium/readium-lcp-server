import { NgModule }                 from '@angular/core';
import { CommonModule }             from '@angular/common';
import { RouterModule }             from '@angular/router';
import {
    FormsModule,
    ReactiveFormsModule }           from '@angular/forms';

import { Ng2DatetimePickerModule }  from 'ng2-datetime-picker';

import { PurchaseService }          from './purchase.service';
import { PurchaseRoutingModule }    from './purchase-routing.module';
import { PurchaseListComponent }    from './purchase-list.component';
import { PurchaseFormComponent }    from './purchase-form.component';
import { PurchaseEditComponent }    from './purchase-edit.component';
import { PurchaseStatusComponent }  from './purchase-status.component';
import { PurchaseAddComponent }     from './purchase-add.component';
import { SortModule }               from '../shared/pipes/sort.module';


@NgModule({
    imports: [
        CommonModule,
        RouterModule,
        FormsModule,
        ReactiveFormsModule,
        PurchaseRoutingModule,
        Ng2DatetimePickerModule,
        SortModule
    ],
    declarations: [
        PurchaseListComponent,
        PurchaseFormComponent,
        PurchaseEditComponent,
        PurchaseStatusComponent,
        PurchaseAddComponent
    ],
    providers: [
        PurchaseService
    ]
})

export class PurchaseModule { }
