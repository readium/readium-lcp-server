import { Component, OnInit }    from '@angular/core';
import { Publication }          from './publication';
import { PublicationService }   from './publication.service';
import { Pipe } from "@angular/core";

@Component({
    moduleId: module.id,
    selector: 'lcp-publication-list',
    templateUrl: 'publication-list.component.html',
})

export class PublicationListComponent implements OnInit {
    publications: Publication[];
    search: string = "";
    order: string;
    reverse: boolean = false;

    constructor(private publicationService: PublicationService) {
        this.publications = [];
        this.order = "title";
    }

    refreshPublications(): void {
        this.publicationService.list().then(
            publications => {
                this.publications = publications;
            }
        );
    }

    orderBy(newOrder: string)
    {
      if (newOrder == this.order)
      {
        this.reverse = !this.reverse;
      }
      else
      {
        this.reverse = false;
        this.order = newOrder
      }
    }

    keptWithFilter (pub :{title: string}): boolean
    {
        if (pub.title.toUpperCase().includes(this.search.toUpperCase()))
        {
            return true;
        }

        return false;
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
