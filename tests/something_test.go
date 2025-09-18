package tests

func (s *IntegrationTestSuite) TestSomething() {
	s.T().Log("db is not nil: ", s.DB() != nil)
}
