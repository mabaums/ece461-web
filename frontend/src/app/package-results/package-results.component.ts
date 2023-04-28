import { Component, Input, OnInit } from '@angular/core';
import { MatSnackBar } from '@angular/material/snack-bar';
import { ActivatedRoute, ParamMap, Route, Router } from '@angular/router';
import { DefaultService, ModelPackage, PackageMetadata } from 'generated';
import { PackageQuery, PackageData} from 'generated';
import { Buffer } from 'buffer';
@Component({
  selector: 'app-package-results',
  templateUrl: './package-results.component.html',
  styleUrls: ['./package-results.component.scss']
})
export class PackageResultsComponent implements OnInit {
  @Input() name?: string;
  @Input() version?: string;
  @Input() regex?: string;

  offset: string = ""

  packages!: PackageMetadata[];


  constructor (private route: ActivatedRoute, private service: DefaultService, private _snackbar: MatSnackBar, private router: Router) {}

  ngOnInit(): void {
    this.searchByNameVersion()
  }
  searchByNameVersion() {
    this._snackbar.open("Searching...", "ok", )
    var query:PackageQuery
    if (this.name || this.version) {
      query = {Name: this.name!, Version: this.version}
    } else {
      query = {Name: "*"}
    }
 
    var queries = [query]

    this.service.packagesList(queries, "", this.offset, "response").subscribe(response => {
      this.packages = response.body!;
      this.offset = response.headers.get("offset")!;
      console.log(this.packages);
      this._snackbar.dismiss()
    }, error => {
      this._snackbar.open(error.message)
    })

  }

  rate(id: string, name: string) {
    this.router.navigate(['/package'], {queryParams : {id: id, name: name}})
  }


  deleteByName(name: string) {
    if (confirm("Are you sure you want to delete all packages with that name?")) {
      this.service.packageByNameDelete("", name).subscribe(body => {
        this.packages = this.packages.filter(item => item.Name != name)
      }, error => {
        this._snackbar.open(error.message, "ok")
      })
    }
  }


  download(id: string) {
    this._snackbar.open("Downloading...")
    this.service.packageRetrieve("", id).subscribe(body => {
      const data = Buffer.from(body.data.Content!, 'base64').toString('binary')
      var decodeData = new Array(data.length);
      for (let i = 0; i < data.length; i++) {
        decodeData[i] = data.charCodeAt(i)
      }
      const blob = new Blob([new Uint8Array(decodeData)], { 
        type: 'application/zip'
      });
      const url = window.URL.createObjectURL(blob)
      window.open(url)
      this._snackbar.dismiss()
    }, error => {
      this._snackbar.open(error.message, "ok")
    })
  }

  update(pkg: PackageMetadata) {
    this.router.navigate(['/update'], {queryParams: {name: pkg.Name, version: pkg.Version, id: pkg.ID}})
  }


  delete(id: string) {
    if (confirm("Are you sure you want to delete?")) {
      this.service.packageDelete("", id).subscribe(body => {
        console.log("Deleting: ", id)
        console.log("Reponse: ", id)
        this.packages = this.packages.filter(item => item.ID != id)
      }, error => {
        this._snackbar.open(error.message, "ok")
      })
    }

  }

  //TODO: LATER
  searchByRegex(regex: string) {
  }
}

