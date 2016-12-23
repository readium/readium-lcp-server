import { NgModule }                 from '@angular/core';
import { CommonModule }             from '@angular/common';
import { RouterModule }             from '@angular/router';

import { DashboardRoutingModule }   from './dashboard-routing.module';
import { DashboardComponent }       from './dashboard.component';

@NgModule({
    imports: [
        CommonModule,
        RouterModule,
        DashboardRoutingModule
    ],
    declarations: [
        DashboardComponent
    ]
})

export class DashboardModule { }
