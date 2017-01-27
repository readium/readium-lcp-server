import { Component, OnDestroy }     from '@angular/core';
import { Subscription }             from 'rxjs/Subscription';

import { SidebarService }           from './shared/sidebar/sidebar.service';

@Component({
    moduleId: module.id,
    selector: 'lcp-app',
    templateUrl: 'app.component.html'
})

export class AppComponent implements OnDestroy {
    sidebarOpen: boolean = false;
    private sidebarSubscription: Subscription;

    constructor(private sidebarService: SidebarService) {
        this.sidebarSubscription = sidebarService.open$.subscribe(
            sidebarOpen => {
                this.sidebarOpen = sidebarOpen;
            }
        );
    }

    ngOnDestroy() {
        // prevent memory leak when component destroyed
        this.sidebarSubscription.unsubscribe();
    }
}
