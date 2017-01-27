import {
    Component,
    Input,
    OnInit }                        from '@angular/core';
import { Router }                   from '@angular/router';
import {
    FormGroup,
    FormControl,
    Validators,
    FormBuilder }                   from '@angular/forms';

import * as moment from 'moment';

import { Purchase }                 from './purchase';
import { PurchaseService }          from './purchase.service';
import { UserService }              from '../user/user.service';
import { User }                     from '../user/user';
import { PublicationService }       from '../publication/publication.service';
import { Publication }              from '../publication/publication';

@Component({
    moduleId: module.id,
    selector: 'lcp-purchase-form',
    templateUrl: 'purchase-form.component.html'
})

export class PurchaseFormComponent implements OnInit{
    @Input()
    purchase: Purchase;
    availablePublications: Publication[];
    availableUsers: User[];

    edit: boolean = false;
    submitButtonLabel: string = "Add";
    form: FormGroup;

    private submitted = false;

    constructor(
        private fb: FormBuilder,
        private router: Router,
        private purchaseService: PurchaseService,
        private userService: UserService,
        private publicationService: PublicationService
        ) {}

    refreshAvailablePublications(): void {
        this.publicationService.list().then(
            publications => {
                this.availablePublications = publications;
            }
        );
    }

    refreshAvailableUsers(): void {
        this.userService.list().then(
            users => {
                this.availableUsers = users;
            }
        );
    }

    ngOnInit(): void {
        this.refreshAvailablePublications();
        this.refreshAvailableUsers();

        if (this.purchase == null) {
            this.purchase = new Purchase();
            this.submitButtonLabel = "Add";
            this.form = this.fb.group({
                "publication": ["", Validators.required],
                "user": ["", Validators.required],
                "end_date": ["", Validators.required],
                "type": ["loan", Validators.required]
            });
        } else {
            this.edit = true;
            this.submitButtonLabel = "Save";
            this.form = this.fb.group({
                "end_date": ["", Validators.required]
            });
        }
    }

    gotoList() {
        this.router.navigate(['/purchases']);
    }

    onCancel() {
        this.gotoList();
    }

    onSubmit() {
        this.bindForm();

        if (this.edit) {
            this.purchaseService.update(
                this.purchase
            ).then(
                purchase => {
                    this.gotoList();
                }
            );
        } else {
            this.purchaseService.add(this.purchase).then(
                purchase => {
                    this.gotoList();
                }
            );
        }

        this.submitted = true;
    }

    // Bind form to purchase
    bindForm(): void {
        let publicationId = this.form.value['publication'];
        let userId = this.form.value['user'];
        let publication = new Publication();
        let user = new User();
        publication.id = publicationId;
        user.id = userId;
        this.purchase.publication = publication;
        this.purchase.user = user;
        this.purchase.type = this.form.value['type'];
        this.purchase.endDate = moment(this.form.value['end_date']).format();
    }
}
