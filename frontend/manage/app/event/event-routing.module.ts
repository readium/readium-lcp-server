import { NgModule }             from '@angular/core';
import { RouterModule, Routes } from '@angular/router';

import { EventComponent }       from './event.component';

const eventRoutes: Routes = [
    { path: 'events', component: EventComponent }
];

@NgModule({
  imports: [
    RouterModule.forChild(eventRoutes)
  ],
  exports: [
    RouterModule
  ]
})

export class EventRoutingModule { }
