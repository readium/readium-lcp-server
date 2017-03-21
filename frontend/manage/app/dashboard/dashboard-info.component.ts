import { Component } from '@angular/core';
import { DashboardInfo, DashboardBestSeller } from './dashboardinfo';
import { DashboardService }   from './dashboard.service';

@Component({
    moduleId: module.id,
    selector: 'lcp-frontend-dashboard-info',
    templateUrl: './dashboard-info.component.html'
})

export class DashboardInfoComponent { 
    infos: DashboardInfo;
    bestSellers: DashboardBestSeller[];

    ngOnInit(): void {
        this.refreshInfos();
    }

    constructor(private dashboardService: DashboardService) {
        
    }

    refreshInfos()
    {
        this.dashboardService.get().then(
            infos => {
                this.infos = infos;
            }
        );
        this.dashboardService.getBestSeller().then(
            bestSellers => {
                this.bestSellers = bestSellers;
            }
        );
             
    }
}