import { NgModule }                 from '@angular/core';
import { CommonModule }             from '@angular/common';
import { RouterModule }             from '@angular/router';

import { EventService }             from './event.service';
import { EventRoutingModule }       from './event-routing.module';
import { EventComponent }           from './event.component';
import { EventInfoComponent }       from './event-info.component';

@NgModule({
    imports: [
        CommonModule,
        RouterModule,
        EventRoutingModule
    ],
    declarations: [
        EventComponent,
        EventInfoComponent
    ],
    providers: [
        EventService
    ]
})

export class EventModule { }
