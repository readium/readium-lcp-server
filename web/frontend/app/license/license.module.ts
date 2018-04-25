import { NgModule }                 from '@angular/core';
import { CommonModule }             from '@angular/common';
import { RouterModule }             from '@angular/router';

import { LicenseService }             from './license.service';
import { LicenseRoutingModule }       from './license-routing.module';
import { LicenseComponent }           from './license.component';
import { LicenseInfoComponent }       from './license-info.component';

import { SortModule }                 from '../shared/pipes/sort.module';

@NgModule({
    imports: [
        CommonModule,
        RouterModule,
        LicenseRoutingModule,
        SortModule
    ],
    declarations: [
        LicenseComponent,
        LicenseInfoComponent
    ],
    providers: [
        LicenseService
    ]
})

export class LicenseModule { }
