import { Component, OnInit }        from '@angular/core';
import { ActivatedRoute, Params }   from '@angular/router';
import 'rxjs/add/operator/switchMap';

import { Publication }              from './publication';
import { PublicationService }       from './publication.service';

@Component({
    moduleId: module.id,
    selector: 'lcp-publication-edit',
    templateUrl: 'publication-edit.component.html'
})

export class PublicationEditComponent implements OnInit {
    publication: Publication;

    constructor(
        private route: ActivatedRoute,
        private publicationService: PublicationService) {
    }

    ngOnInit(): void {
        this.route.params
            .switchMap((params: Params) => this.publicationService.get(""+params['id']))
            .subscribe(publication => {
                this.publication = publication
            });
    }
}
