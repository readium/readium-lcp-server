import { Component, Input } from '@angular/core';
import { License } from './license';
import { LicenseService }   from './license.service';

declare var Config: any;

@Component({
    moduleId: module.id,
    selector: 'lcp-frontend-license-info',
    templateUrl: './license-info.component.html'
})


export class LicenseInfoComponent {
    @Input('filterBox') filterBox: any;

    licenses: License[];
    filter: number = 0;
    filtred = false;
    baseUrl: string;

    reverse: boolean = false;
    order: string;



    ngOnInit(): void {
        this.baseUrl = Config.frontend.url;
        this.refreshInfos();

        this.order = "publicationTitle";
    }

    constructor(private licenseService: LicenseService) {

    }

    onSubmit(){
        this.filtred = true;
        this.refreshInfos();
    }

    refreshInfos()
    {
        this.licenseService.get(this.filter).then(
            infos => {
                this.licenses = infos;
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

    keyPressed(key:number)
    {
        if (key == 13)
        {
            this.onSubmit()
        }
    }
}
