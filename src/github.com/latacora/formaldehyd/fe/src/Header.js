import React, { Component } from 'react';
import logo from './logo.svg';
import './App.css';

import 'grommet-css';

import App from 'grommet/components/App';
import Split from 'grommet/components/Split';
import Header from 'grommet/components/Header';
import Search from 'grommet/components/Search';
import Menu from 'grommet/components/Menu';
import Anchor from 'grommet/components/Anchor';
import Title from 'grommet/components/Title';
import Box from 'grommet/components/Box';
import Actions from 'grommet/components/icons/base/Actions';

class AppHeader extends Component {
  render() {
    return (
      <Header>
        <Title>
          Sample Title
        </Title>
        <Box flex={ true }
             justify='end'
             direction='row'
             responsive={ false }>
          <Search inline={ true }
                  fill={ true }
                  size='medium'
                  placeHolder='Search'
                  dropAlign={ {"right": "right"} } />
          <Menu icon={ <Actions /> }
                dropAlign={ {"right": "right"} }>
            <Anchor href='#'
                    className='active'>
              First
            </Anchor>
            <Anchor href='#'>
              Second
            </Anchor>
          </Menu>
        </Box>
      </Header>
      );
  }
}

export default AppHeader;
