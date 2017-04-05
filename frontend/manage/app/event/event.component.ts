import { Component } from '@angular/core';
import { EventService } from './event.service'

@Component({
    moduleId: module.id,
    selector: 'lcp-frontend-event',
    templateUrl: 'event.component.html',
    providers: [EventService]
})

export class EventComponent { }
