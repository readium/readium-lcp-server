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
import { PurchaseAddComponent }     from './purchase-add.component';

@NgModule({
    imports: [
        CommonModule,
        RouterModule,
        FormsModule,
        ReactiveFormsModule,
        PurchaseRoutingModule,
        Ng2DatetimePickerModule
    ],
    declarations: [
        PurchaseListComponent,
        PurchaseFormComponent,
        PurchaseAddComponent
    ],
    providers: [
        PurchaseService
    ]
})

export class PurchaseModule { }
