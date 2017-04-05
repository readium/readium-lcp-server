import { Component } from '@angular/core';
import { Event } from './event';
import { EventService }   from './event.service';

@Component({
    moduleId: module.id,
    selector: 'lcp-frontend-event-info',
    templateUrl: './event-info.component.html'
})

export class EventInfoComponent { 
    infos: Event;

    ngOnInit(): void {
        this.refreshInfos();
    }

    constructor(private eventService: EventService) {
        
    }

    refreshInfos()
    {
        this.eventService.get().then(
            infos => {
                this.infos = infos;
            }
        );             
    }
}