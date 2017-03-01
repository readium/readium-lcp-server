import { Component, OnInit }    from '@angular/core';
import { Publication }          from './publication';
import { PublicationService }   from './publication.service';

@Component({
    moduleId: module.id,
    selector: 'lcp-publication-list',
    templateUrl: 'publication-list.component.html'
})

export class PublicationListComponent implements OnInit {
    publications: Publication[];

    constructor(private publicationService: PublicationService) {
        this.publications = [];
    }

    refreshPublications(): void {
        this.publicationService.list().then(
            publications => {
                this.publications = publications;
            }
        );
    }

    ngOnInit(): void {
        this.refreshPublications();
    }

    onRemove(objId: any): void {
        this.publicationService.delete(objId).then(
            publication => {
                this.refreshPublications();
            }
        );
    }
 }
