/**
 * System configuration for Angular samples
 * Adjust as necessary for your application needs.
 */
(function (global) {
  System.config({
    paths: {
      // paths serve as alias
      'npm:': 'node_modules/'
    },
    // map tells the System loader where to look for things
    map: {
      // our app is within the app folder
      app: 'dist/app',

      // angular bundles
      '@angular/core': 'npm:@angular/core/bundles/core.umd.js',
      '@angular/common': 'npm:@angular/common/bundles/common.umd.js',
      '@angular/compiler': 'npm:@angular/compiler/bundles/compiler.umd.js',
      '@angular/platform-browser': 'npm:@angular/platform-browser/bundles/platform-browser.umd.js',
      '@angular/platform-browser-dynamic': 'npm:@angular/platform-browser-dynamic/bundles/platform-browser-dynamic.umd.js',
      '@angular/http': 'npm:@angular/http/bundles/http.umd.js',
      '@angular/router': 'npm:@angular/router/bundles/router.umd.js',
      '@angular/forms': 'npm:@angular/forms/bundles/forms.umd.js',

      // other libraries
      'rxjs':                      'npm:rxjs',
      'angular-in-memory-web-api': 'npm:angular-in-memory-web-api/bundles/in-memory-web-api.umd.js',
      'jssha': 'npm:jssha/src',
      'ng2-datetime-picker': 'npm:ng2-datetime-picker/dist',
      'moment': 'node_modules/moment',
      'config': '/config.js',
      'file-saver': 'npm:file-saver'
    },
    // packages tells the System loader how to load when no filename and/or no extension
    packages: {
      app: {
        main: './main.js',
        defaultExtension: 'js'
      },
      rxjs: {
        defaultExtension: 'js'
      },
      'jssha': {
        main: 'sha.js',
        defaultExtension: 'js'

      },
      'ng2-datetime-picker': {
        main: 'ng2-datetime-picker.umd.js',
        defaultExtension: 'js'
      },
      'moment': {
        main: 'moment.js',
        defaultExtension: 'js'
      },
      'file-saver': {
        main: './FileSaver.js',
        defaultExtension: 'js'
      }
    }
  });
})(this);
