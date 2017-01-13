import { Injectable }    from '@angular/core';
import { Http, Headers }    from '@angular/http';

import 'rxjs/add/operator/toPromise';

import { CrudItem }         from './crud-item'

export abstract class CrudService<T extends CrudItem> {
    http: Http;
    defaultHttpHeaders = new Headers(
        {'Content-Type': 'application/json'});
    baseUrl: string;

    // Decode Json from API and build crud object
    abstract decode(jsonObj: any): T;

    // Encode crud object to API json
    abstract encode(obj: T): any;

    list(): Promise<T[]> {
        var self = this
        return this.http.get(
                this.baseUrl,
                { headers: this.defaultHttpHeaders })
            .toPromise()
            .then(function (response) {
                let items: T[] = [];

                for (let jsonObj of response.json()) {
                    items.push(self.decode(jsonObj));
                }

                return items;
            })
            .catch(this.handleError);
    }

    get(id: string): Promise<T> {
        var self = this
        return this.http
            .get(
                this.baseUrl + "/" + id,
                { headers: this.defaultHttpHeaders })
            .toPromise()
            .then(function (response) {
                let jsonObj = response.json();
                return self.decode(jsonObj);
            })
            .catch(this.handleError);
    }

    delete(id: string): Promise<boolean> {
        var self = this
        return this.http.delete(this.baseUrl + "/" + id)
            .toPromise()
            .then(function (response) {
                if (response.ok) {
                    return true;
                } else {
                    throw 'Error creating user ' + response.text;
                }
            })
            .catch(this.handleError);
    }

    add(obj: T): Promise<T> {
        return this.http
            .post(
                this.baseUrl,
                this.encode(obj),
                { headers: this.defaultHttpHeaders })
            .toPromise()
            .then(function (response) {
                if (response.ok) {
                    return obj;
                } else {
                    throw 'Error creating user ' + response.text;
                }
            })
            .catch(this.handleError);
    }

    update(obj: T): Promise<T> {
        return this.http
            .put(
                this.baseUrl + "/" + obj.id,
                this.encode(obj),
                { headers: this.defaultHttpHeaders })
            .toPromise()
            .then(function (response) {
                if (response.ok) {
                    return obj;
                } else {
                    throw 'Error creating user ' + response.text;
                }
            })
            .catch(this.handleError);
    }

    protected handleError(error: any): Promise<any> {
        console.error('An error occurred', error);
        return Promise.reject(error.message || error);
    }
}
