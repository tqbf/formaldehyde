import React, { Component } from 'react';
import PropTypes from 'prop-types';
import logo from './logo.svg';
import './App.css';
import BaseComponent from './BaseComponent';

import 'grommet-css';

import Select from 'grommet/components/Select';
import Box from 'grommet/components/Box';

export default class SelectField extends BaseComponent {
  state = {
  }

  constructor(props) {
    super(props);
  }

  render() {
    return (
      <Box pad={ {horizontal: this.props.hpad, vertical: this.props.vpad, between: this.props.bpad} }>
        <legend>
          { this.props.legend }
        </legend>
        <Select placeHolder='None'
                inline={ false }
                multiple={ false }
                options={ this.props.options }
                value={ undefined }
                onChange={ this.props.onChange } />
      </Box>
      );
  }
}

SelectField.propTypes = {
  legend: PropTypes.string,
  options: PropTypes.array.isRequired,
  onChange: PropTypes.func.isRequired,
};

SelectField.defaultProps = {
  vpad: "small",
  hpad: "medium",
  bpad: "none",
};
