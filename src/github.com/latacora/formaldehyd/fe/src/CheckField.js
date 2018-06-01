import React, { Component } from 'react';
import PropTypes from 'prop-types';
import logo from './logo.svg';
import './App.css';
import BaseComponent from './BaseComponent';

import 'grommet-css';

import FormField from 'grommet/components/FormField';
import CheckBox from 'grommet/components/CheckBox';
import Box from 'grommet/components/Box';

export default class CheckField extends BaseComponent {
  constructor(props) {
    super(props);
  }

  render() {
    let checked = this.props.checked;
    if (checked == "on") {
      checked = true;
    }

    return (
      <Box pad={ {horizontal: this.props.hpad, vertical: this.props.vpad, between: this.props.bpad} }>
        <CheckBox label={ this.props.label }
                  checked={ checked }
                  toggle={ this.props.toggle }
                  onChange={ this.props.onChange }
                  reverse={ this.props.reverse } />
      </Box>
      );
  }
}

CheckField.propTypes = {
  label: PropTypes.string.isRequired,
  toggle: PropTypes.bool,
  reverse: PropTypes.bool,
  onChange: PropTypes.func.isRequired,
  vpad: PropTypes.oneOf(['none', 'small', 'medium', 'large']),
  hpad: PropTypes.oneOf(['none', 'small', 'medium', 'large']),
  bpad: PropTypes.oneOf(['none', 'small', 'medium', 'large']),
};

CheckField.defaultProps = {
  vpad: "small",
  hpad: "medium",
  bpad: "none",
  toggle: false,
  reverse: false,
  checked: false,
};
