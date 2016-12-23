import { Component, Input, OnInit }     from '@angular/core';
import { Router }                       from '@angular/router';
import {
    FormGroup,
    FormControl,
    Validators,
    FormBuilder }                       from '@angular/forms';

import { User }                         from './user';
import { UserService }                  from './user.service';

@Component({
    moduleId: module.id,
    selector: 'lcp-user-form',
    templateUrl: 'user-form.component.html'
})

export class UserFormComponent implements OnInit {
    @Input()
    user: User;

    submitButtonLabel: string = "Add";
    form: FormGroup;

    private submitted = false;

    constructor(
        private fb: FormBuilder,
        private router: Router,
        private userService: UserService) {
    }

    ngOnInit(): void {
        if (this.user == null) {
            this.submitButtonLabel = "Add";
            this.form = this.fb.group({
                "alias": ["", Validators.required],
                "email": ["", Validators.required],
                "password": ["", Validators.required]
            });
        } else {
            this.submitButtonLabel = "Save";
            this.form = this.fb.group({
                "alias": [this.user.alias, Validators.required],
                "email": [this.user.email, Validators.required],
                "password": ""
            });
        }
    }

    gotoList() {
        this.router.navigate(['/users']);
    }

    onCancel() {
        this.gotoList();
    }

    onSubmit() {
        if (this.user) {
            // Update user
            // FIXME: Only update password
            this.userService.update(
                this.user,
                this.form.value['password']
            ).then(
                user => {
                    this.gotoList();
                }
            );
        } else {
            // Create user
            this.userService.create(
                this.form.value['alias'],
                this.form.value['email'],
                this.form.value['password']
            ).then(
                user => {
                    this.gotoList();
                }
            );
        }

        this.submitted = true;
    }
}
