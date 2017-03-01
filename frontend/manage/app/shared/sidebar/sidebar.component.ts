import { Component }        from '@angular/core';
import { SidebarService }   from './sidebar.service';

@Component({
    moduleId: module.id,
    selector: 'lcp-sidebar',
    templateUrl: 'sidebar.component.html'
})

export class SidebarComponent {
    constructor(private sidebarService: SidebarService) {
    }

    toggle() {
       this.sidebarService.toggle();
    }
}
