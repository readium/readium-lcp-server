import { NgModule }                 from '@angular/core';
import { CommonModule }             from '@angular/common';
import { RouterModule }             from '@angular/router';

import { DashboardRoutingModule }   from './dashboard-routing.module';
import { DashboardComponent }       from './dashboard.component';
import { DashboardInfoComponent }       from './dashboard-info.component';

@NgModule({
    imports: [
        CommonModule,
        RouterModule,
        DashboardRoutingModule
        
    ],
    declarations: [
        DashboardComponent,
        DashboardInfoComponent
    ]
})

export class DashboardModule { }
