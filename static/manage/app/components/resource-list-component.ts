import { Component, Input, OnInit } from '@angular/core';

import { Router } from '@angular/router';
// import { User } from './user';
import { Resource } from './resource';

import { ResourceService } from './resource.service';


@Component({
    moduleId: module.id,
    selector: 'resources',
    templateUrl: '/app/components/resource-list.html',
    styleUrls: ['/style.css'],
    providers: [ResourceService]
})


export class ResourcesComponent implements OnInit {
    resources: Resource[];
    selectedResource: Resource;
    @Input() id: string;

    constructor(private ResourceService: ResourceService, private router: Router) { }


    getResources(): void {
        this.ResourceService.getResources().then(Resources => this.resources = Resources);
    }

    ngOnInit(): void {
        this.getResources();
    }

    onSelect(resource: Resource): void {
        this.selectedResource = resource;
    }
}
