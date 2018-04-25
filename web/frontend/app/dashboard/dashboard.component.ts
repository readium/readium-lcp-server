import { Component } from '@angular/core';
import { DashboardService } from './dashboard.service'

@Component({
    moduleId: module.id,
    selector: 'lcp-frontend-dashboard',
    templateUrl: 'dashboard.component.html',
    providers: [DashboardService]
})

export class DashboardComponent { }
