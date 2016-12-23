import { NgModule }             from '@angular/core';
import { CommonModule }         from '@angular/common';
import { RouterModule }         from '@angular/router';

import { SidebarService }       from './sidebar.service';
import { SidebarComponent }     from './sidebar.component';

@NgModule({
    imports: [
        CommonModule,
        RouterModule
    ],
    declarations: [
        SidebarComponent
    ],
    exports: [
        SidebarComponent
    ],
    providers: [
        SidebarService
    ]
})

export class SidebarModule { }
