import * as _ from 'underscore';
import $ from 'jquery';
import jQuery from 'jquery';

import 'bootstrap';
import 'bootstrap/dist/css/bootstrap.css';
import 'bootstrap-switch';
import 'bootstrap-switch/dist/css/bootstrap3/bootstrap-switch.css';
import 'font-awesome/css/font-awesome.css'
import '../css/site.css'
import UiCommon from './ui/common.js'
import './ui/menu.js'

import Common from './common.js'
import {Grid} from "ag-grid-community";
import "ag-grid-community/dist/styles/ag-grid.css";
import "ag-grid-community/dist/styles/ag-theme-balham.css";

const gridOptions = {
    columnDefs: [
        {
            headerName: 'App',
            field: 'app'
        },
        {
            headerName: 'Date',
            field: 'date',
            cellRenderer: (data) => {
                return new Date(data.date).format('MM/DD/YYYY HH:mm')
            }
        }
    ],
    columnTypes: {
        "dateColumn": {
            filter: 'agDateColumnFilter',
            suppressMenu:true
        }
    },
    rowData: [
    {
        app: 'Nextcloud', date: 1553201616 }
    ]
};

$( document ).ready(function () {
  let eGridDiv = document.querySelector('#backupGrid');
  
  let grid = new Grid(eGridDiv, gridOptions);
});
